package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/corny/dnscheck/check"
	"github.com/corny/dnscheck/export"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app      = kingpin.New("dnscheck", "A public dns checker")
	debug    = app.Flag("debug", "Enable debug mode.").Bool()
	syslog   = app.Flag("syslog", "Prepare logging for syslog (print to stdout, no timestamps)").Bool()
	database = app.Flag("database", "Path to file containing the database configuration").Default("database.yml").String()

	checkCmd             = app.Command("check", "Run a DNS check")
	metricsListenAddress = checkCmd.Flag("web.listen-address", "Address on which to expose metrics and web interface").Default(":9000").String()
	metricsPath          = checkCmd.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	domains              = checkCmd.Flag("domains", "Path to file containing the domain list").Default("domains.txt").String()
	checker              check.Checker
	finishedWg           sync.WaitGroup

	exportCmd  = app.Command("export", "Export all DNS servers")
	batchSize  = exportCmd.Flag("batch-size", "Batch size for fetching database records").Default("1000").Uint()
	outputPath = exportCmd.Flag("output", "Path to output directory").Default(".").String()

	purgeCmd    = app.Command("purge", "Purge database")
	maxCheckAge = purgeCmd.Flag("max-age", "Maximum age of check results and unreachable nameservers in days").Default("30").Uint()

	dbConn *sql.DB
)

func init() {
	checkCmd.Flag("reference", "The nameserver that every other is compared with").Default("8.8.8.8").StringVar(&checker.ReferenceServer)
	checkCmd.Flag("workers", "Number of worker routines").Default("32").UintVar(&checker.WorkersCount)
	checkCmd.Flag("attempts", "Maximum number of attempts per query on timeouts").Default("3").UintVar(&checker.MaxAttempts)
	checkCmd.Flag("timeout", "Timeout per dns query").Default("3s").DurationVar(&checker.DNSClient.ReadTimeout)
	checkCmd.Flag("geodb", "Path to GeoDB database").Default("GeoLite2-City.mmdb").StringVar(&checker.GeoDbPath)
}

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if *syslog {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
	}

	environment := os.Getenv("RAILS_ENV")
	if environment == "" {
		environment = "development"
	}

	// load database configuration
	connectionPath := databasePath(*database, environment)

	// Open SQL connection
	var err error
	dbConn, err = sql.Open("mysql", connectionPath)
	if err != nil {
		log.Fatalln("cannot connect to database:", err)
	}
	defer dbConn.Close()

	// Run the command
	switch cmd {
	case checkCmd.FullCommand():
		err = runCheck()

	case exportCmd.FullCommand():
		exporter := export.Exporter{
			BatchSize:   *batchSize,
			Debug:       *debug,
			Connection:  dbConn,
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
