package export

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/corny/dnscheck/internal/store"
)

type Exporter struct {
	Destination string // Destination is the output directory
	Debug       bool
	Store       *store.Store
	BatchSize   uint
	queue       map[string]*Writer
}

// Run starts the export and wait until it is finished
func (ex *Exporter) Run() error {
	if ex.queue != nil {
		return errors.New("already started")
	}

	if ex.BatchSize < 1 {
		return errors.New("batch size too small")
	}

	if ex.Store == nil {
		return errors.New("store missing")
	}

	if _, err := os.Stat(ex.Destination); err != nil {
		return fmt.Errorf("invalid destination %q: %w", ex.Destination, err)
	}

	ex.queue = make(map[string]*Writer)
	t0 := time.Now()
	var n uint

	err := ex.Store.EachNameserver(ex.BatchSize, func(ns *store.Nameserver) (e error) {
		if e = ex.exportEntry("nameservers-all", ns); e != nil {
			return
		}
		if ns.IsValid() {
			if e = ex.exportEntry("nameservers", ns); e != nil {
				return
			}
			if ns.Country != nil && *ns.Country != "" {
				if e = ex.exportEntry(strings.ToLower(*ns.Country), ns); e != nil {
					return
				}
			}
		}
		n++
		return nil
	})

	if err != nil {
		panic(err)
	}

	for name, q := range ex.queue {
		if err = q.Close(); err != nil {
			panic(err)
		}
		if ex.Debug || strings.HasPrefix(name, "nameservers") {
			log.Printf("%s = %d", name, q.Count())
		}
	}
	if ex.Debug {
		log.Printf("total = %d", n)
	}
	log.Printf("time = %v", time.Since(t0))

	return nil
}

func (ex *Exporter) exportEntry(channel string, ns *store.Nameserver) (err error) {
	q, found := ex.queue[channel]
	if !found {
		q, err = NewWriter(ex.Destination, channel)
		if err != nil {
			return
		}
		ex.queue[channel] = q
	}

	q.Channel <- ns
	return nil
}
