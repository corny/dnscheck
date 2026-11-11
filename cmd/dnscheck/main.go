package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kingpin/v2"
	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/checkrun"
	"github.com/corny/dnscheck/internal/export"
	"github.com/corny/dnscheck/internal/store"
)

var (
	app      = kingpin.New("dnscheck", "A public dns checker")
	debug    = app.Flag("debug", "Enable debug mode.").Bool()
	syslog   = app.Flag("syslog", "Prepare logging for syslog (print to stdout, no timestamps)").Bool()
	database = app.Flag("database", "Data source name (DSN) for the PostgreSQL database, defaults to environment variable DSN").Default(defaultDSN()).String()

	checkCmd      = app.Command("check", "Run DNS checks continuously")
	listenAddress = checkCmd.Flag("web.listen-address", "Listening address for the web interface").Default(":8000").String()
	metricsPath   = checkCmd.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	allowOrigins  = checkCmd.Flag("web.allow-origin", "Allowed origins for CORS").Default("http://localhost:3000").Strings()
	domains       = checkCmd.Flag("domains", "Path to file containing the domain list").Default("domains.txt").String()
	checkInterval = checkCmd.Flag("interval", "Check interval").Default("1h").Duration()
	checker       check.Checker

	exportCmd  = app.Command("export", "Export all DNS servers")
	batchSize  = exportCmd.Flag("batch-size", "Batch size for fetching database records").Default("1000").Uint()
	outputPath = exportCmd.Flag("output", "Path to output directory").Default(".").String()

	purgeCmd    = app.Command("purge", "Purge database")
	maxCheckAge = purgeCmd.Flag("max-age", "Maximum age of check results and unreachable nameservers in days").Default("30").Uint()

	db *store.Store
)

func defaultDSN() string {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		dsn = "host=/var/run/postgresql dbname=publicdns"
	}

	return dsn
}

func init() {
	checkCmd.Flag("reference", "The nameserver that every other is compared with").Default("8.8.8.8").StringVar(&checker.ReferenceServer)
	checkCmd.Flag("workers", "Number of worker routines").Default("32").UintVar(&checker.WorkersCount)
	checkCmd.Flag("attempts", "Maximum number of attempts per query on timeouts").Default("3").UintVar(&checker.MaxAttempts)
	checkCmd.Flag("timeout", "Timeout per dns query").Default("3s").DurationVar(&checker.DNSClient.ReadTimeout)
	checkCmd.Flag("geodb-city", "Path to GeoDB city database").Default("/var/lib/GeoIP/GeoLite2-City.mmdb").StringVar(&checker.GeoDbPathCity)
	checkCmd.Flag("geodb-asn", "Path to GeoDB asn database").Default("/var/lib/GeoIP/GeoLite2-ASN.mmdb").StringVar(&checker.GeoDbPathASN)
}

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *syslog {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
	} else {
		log.SetFlags(log.Lshortfile)
	}

	// Open SQL connection
	var err error
	db, err = store.Open(*database)
	if err != nil {
		log.Fatalln("cannot connect to database:", err)
	}
	defer db.Close()

	if err := db.EnsureSchema(); err != nil {
		log.Fatalln("cannot create database schema:", err)
	}

	// Run the command
	switch cmd {
	case checkCmd.FullCommand():
		startHTTP()

		runner := checkrun.New(&checker, db, *domains, *checkInterval, *debug)
		go func() {
			err := runner.Run()
			if err != nil {
				log.Panicln(err)
			}
		}()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		log.Printf("received %v, shutting down", <-sigs)

		runner.Stop()

	case exportCmd.FullCommand():
		exporter := export.Exporter{
			BatchSize:   *batchSize,
			Debug:       *debug,
			Store:       db,
			Destination: *outputPath,
		}
		err = exporter.Run()

	case purgeCmd.FullCommand():
		err = runPurge()
	}

	// Check result
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
