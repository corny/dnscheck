package geoip

import (
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"sync"
	"time"

	geoip2 "github.com/oschwald/geoip2-golang/v2"
)

// reloadInterval is the interval in which the database file is checked for
// changes.
var reloadInterval = time.Minute

// Database is a wrapper around the GeoIP2 database
type Database struct {
	path     string
	interval time.Duration // see reloadInterval
	reader   *geoip2.Reader
	modTime  time.Time
	mtx      sync.RWMutex
	shutdown chan struct{}
}

// New opens a GeoIP database
func New(path string) (*Database, error) {
	db := &Database{
		path:     path,
		interval: reloadInterval,
	}

	if err := db.open(); err != nil {
		return nil, err
	}

	db.shutdown = make(chan struct{})
	go db.watcher()
	return db, nil
}

// City looks up the city
func (db *Database) City(ip net.IP) (*geoip2.City, error) {
	addr, err := toAddr(ip)
	if err != nil {
		return nil, err
	}

	db.mtx.RLock()
	defer db.mtx.RUnlock()

	return db.reader.City(addr)
}

// ASN looks up the ASN
func (db *Database) ASN(ip net.IP) (*geoip2.ASN, error) {
	addr, err := toAddr(ip)
	if err != nil {
		return nil, err
	}

	db.mtx.RLock()
	defer db.mtx.RUnlock()

	return db.reader.ASN(addr)
}

func toAddr(ip net.IP) (netip.Addr, error) {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Addr{}, fmt.Errorf("invalid IP address: %v", ip)
	}
	return addr.Unmap(), nil
}

// open (re)opens the database
func (db *Database) open() error {
	db.mtx.Lock()
	defer db.mtx.Unlock()

	info, err := os.Stat(db.path)
	if err != nil {
		return err
	}

	reader, err := geoip2.Open(db.path)
	if err != nil {
		return err
	}

	if db.reader != nil {
		db.reader.Close()
	}

	db.reader = reader
	db.modTime = info.ModTime()
	return nil
}

// Close closes the database
func (db *Database) Close() error {
	close(db.shutdown)
	return nil
}

// watcher runs check() periodically until shutdown
func (db *Database) watcher() {
	ticker := time.NewTicker(db.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := db.check(); err != nil {
				slog.Error("GeoIP database reload failed", "path", db.path, "error", err)
			}
		case <-db.shutdown:
			db.mtx.Lock()
			db.reader.Close()
			db.mtx.Unlock()
			return
		}
	}
}

// check checks the database for modifications and reload it if necessary
func (db *Database) check() error {
	info, err := os.Stat(db.path)
	if err != nil {
		return err
	}
	if info.ModTime() != db.modTime {
		slog.Info("reloading GeoIP database", "path", db.path)
		return db.open()
	}
	return nil
}
