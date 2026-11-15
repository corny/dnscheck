package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/corny/dnscheck/internal/check"
	"github.com/corny/dnscheck/internal/checkrun"
	"github.com/corny/dnscheck/internal/export"
	"github.com/corny/dnscheck/internal/geoip"
	"github.com/corny/dnscheck/internal/store"
	"github.com/corny/dnscheck/internal/web"
)

var (
	app      = kingpin.New("dnscheck", "A public dns checker")
	debug    = app.Flag("debug", "Enable debug mode.").Bool()
	database = app.Flag("database", "Data source name (DSN) for the PostgreSQL database, defaults to environment variable DSN").Default(defaultDSN()).String()

	serveCmd      = app.Command("serve", "Serve the web pages and run checks, export and purge continuously")
	listenAddress = serveCmd.Flag("web.listen-address", "Listening address for the web interface").Default(":8000").String()
	metricsPath   = serveCmd.Flag("web.telemetry-path", "Path under which to expose metrics").Default("/metrics").String()
	domains       = serveCmd.Flag("domains", "Path to file containing the domain list").Default("domains.txt").String()
	checkInterval = serveCmd.Flag("interval", "Check interval").Default("1h").Duration()
	dataDir       = serveCmd.Flag("data-dir", "Data directory with the subdirectories export (export files) and geoip (GeoLite2 databases, downloaded if MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY are set)").Default("/data").String()
	exportEvery   = serveCmd.Flag("export.interval", "Export interval").Default("4h").Duration()
	purgeEvery    = serveCmd.Flag("purge.interval", "Purge interval").Default("24h").Duration()
	purgeMaxAge   = serveCmd.Flag("purge.max-age", "Maximum age of check results and unreachable nameservers in days").Default("30").Uint()
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
	serveCmd.Flag("reference", "The nameserver that every other is compared with").Default("8.8.8.8").StringVar(&checker.ReferenceServer)
	serveCmd.Flag("workers", "Number of worker routines").Default("32").UintVar(&checker.WorkersCount)
	serveCmd.Flag("attempts", "Maximum number of attempts per query on timeouts").Default("3").UintVar(&checker.MaxAttempts)
	serveCmd.Flag("timeout", "Timeout per dns query").Default("3s").DurationVar(&checker.DNSClient.ReadTimeout)
}

func main() {
	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	slog.SetDefault(newLogger(os.Stderr, *debug))

	// Open SQL connection
	var err error
	db, err = store.Open(*database)
	if err != nil {
		fatal("cannot connect to database", "error", err)
	}
	defer db.Close()

	if err := db.EnsureSchema(); err != nil {
		fatal("cannot create database schema", "error", err)
	}

	// Run the command
	switch cmd {
	case serveCmd.FullCommand():
		exportDir := filepath.Join(*dataDir, "export")
		geoipDir := filepath.Join(*dataDir, "geoip")

		if err := checker.ReadDomains(*domains); err != nil {
			fatal("cannot read domain list", "error", err)
		}

		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		startHTTP(web.Config{
			Store:     db,
			Checker:   &checker,
			ExportDir: exportDir,
			Domains:   checker.Domains(),
			Reference: checker.ReferenceServer,
			Timeout:   checker.DNSClient.ReadTimeout,
			PurgeDays: *purgeMaxAge,
		})

		var tasks sync.WaitGroup
		every(ctx, &tasks, *exportEvery, "export", func() error { return runExport(exportDir, 1000) })
		every(ctx, &tasks, *purgeEvery, "purge", func() error { return runPurge(*purgeMaxAge) })

		updater := &geoip.Updater{
			Dir:        geoipDir,
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
			slog.Info("MAXMIND_ACCOUNT_ID or MAXMIND_LICENSE_KEY not set, using existing GeoLite2 databases", "dir", geoipDir)
		}

		if ctx.Err() == nil {
			runner := checkrun.New(&checker, db, *checkInterval)
			go func() {
				if err := runner.Run(); err != nil {
					fatal("checker stopped", "error", err)
				}
			}()

			<-ctx.Done()
			slog.Info("shutting down")
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
