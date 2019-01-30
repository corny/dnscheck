package geoip

import (
	"log"
	"net"
	"os"
	"sync"
	"time"

	geoip2 "github.com/oschwald/geoip2-golang"
)

// Database is a wrapper around the GeoIP2 database
type Database struct {
	path     string
	reader   *geoip2.Reader
	modTime  time.Time
	mtx      sync.RWMutex
	shutdown chan struct{}
}

// New opens a GeoIP database
func New(path string) (*Database, error) {
	db := &Database{
		path: path,
	}

	if err := db.open(); err != nil {
		return nil, err
	}

	db.shutdown = make(chan struct{})
	go db.watcher()
	return db, nil
}

// City looks up ISO code and City by IP address
func (db *Database) City(ip net.IP) (isocode string, city string, err error) {
	db.mtx.RLock()
	record, err := db.reader.City(ip)
	db.mtx.RUnlock()

	if err != nil {
		return "", "", err
	}

	return record.Country.IsoCode, record.City.Names["en"], nil
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

// watcher runs periodically check() until shutdown
func (db *Database) watcher() {
	select {
	case <-time.After(time.Minute):
		db.check()
	case <-db.shutdown:
		db.mtx.Lock()
		db.reader.Close()
		db.mtx.Unlock()
		return
	}
}

// check checks the database for modifications and reload it if necessary
func (db *Database) check() error {
	info, err := os.Stat(db.path)
	if err != nil {
		return err
	}
	if info.ModTime() != db.modTime {
		log.Println("GeoIP database reloading")
		return db.open()
	}
	return nil
}
