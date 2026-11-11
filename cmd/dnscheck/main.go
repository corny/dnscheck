package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/checkrun"
	"github.com/corny/dnscheck/internal/export"
	"github.com/corny/dnscheck/internal/geoip"
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
	exportOutput  = checkCmd.Flag("export.output", "Path to the export directory").Default(".").String()
	exportEvery   = checkCmd.Flag("export.interval", "Export interval").Default("4h").Duration()
	purgeEvery    = checkCmd.Flag("purge.interval", "Purge interval").Default("24h").Duration()
	purgeMaxAge   = checkCmd.Flag("purge.max-age", "Maximum age of check results and unreachable nameservers in days").Default("30").Uint()
	geoipDir      = checkCmd.Flag("geoip.dir", "Directory of the GeoLite2 databases. They are downloaded if MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY are set.").Default("/var/lib/GeoIP").String()
	checker       check.Checker

	exportCmd  = app.Command("export", "Export all DNS servers")
	batchSize  = exportCmd.Flag("batch-size", "Batch size for fetching database records").Default("1000").Uint()
	outputPath = exportCmd.Flag("output", "Path to output directory").Default(".").String()

	purgeCmd    = app.Command("purge", "Purge database")
	maxCheckAge = purgeCmd.Flag("max-age", "Maximum age of check results and unreachable nameservers in days").Default("30").Uint()

	db *store.Store
)

func runExport(destination string, batchSize uint) error {
	exporter := export.Exporter{
		BatchSize:   batchSize,
		Debug:       *debug,
		Store:       db,
		Destination: destination,
	}
	return exporter.Run()
}

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
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		startHTTP()

		var tasks sync.WaitGroup
		every(ctx, &tasks, *exportEvery, "export", func() error { return runExport(*exportOutput, 1000) })
		every(ctx, &tasks, *purgeEvery, "purge", func() error { return runPurge(*purgeMaxAge) })

		updater := &geoip.Updater{
			Dir:        *geoipDir,
			AccountID:  os.Getenv("MAXMIND_ACCOUNT_ID"),
			LicenseKey: os.Getenv("MAXMIND_LICENSE_KEY"),
			MaxAge:     20 * time.Hour,
			Client:     &http.Client{Timeout: 10 * time.Minute},
		}
		checker.GeoDbPathCity = updater.Path(geoip.EditionCity)
		checker.GeoDbPathASN = updater.Path(geoip.EditionASN)

		if updater.AccountID != "" && updater.LicenseKey != "" {
			if waitForGeoIP(ctx, updater) == nil {
				every(ctx, &tasks, 24*time.Hour, "GeoLite2 update", func() error { return updater.Update(ctx) })
			}
		} else {
			log.Println("MAXMIND_ACCOUNT_ID or MAXMIND_LICENSE_KEY not set, using existing GeoLite2 databases in", *geoipDir)
		}

		if ctx.Err() == nil {
			runner := checkrun.New(&checker, db, *domains, *checkInterval, *debug)
			go func() {
				err := runner.Run()
				if err != nil {
					log.Panicln(err)
				}
			}()

			<-ctx.Done()
			log.Println("shutting down")
			runner.Stop()
		}

		tasks.Wait()

	case exportCmd.FullCommand():
		err = runExport(*outputPath, *batchSize)

	case purgeCmd.FullCommand():
		err = runPurge(*maxCheckAge)
	}

	// Check result
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
