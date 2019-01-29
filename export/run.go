package export

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/erikdubbelboer/gspt"
)

const titleFmt = `"%s (master) %d records/s, current #%d"`

type Exporter struct {
	Destination string // Destination is the output directory
	Debug       bool
	Connection  *sql.DB
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

	if ex.Connection == nil {
		return errors.New("connection missing")
	}

	if _, err := os.Stat(ex.Destination); err != nil {
		return errors.Wrapf(err, "invalid destination '%s'", ex.Destination)
	}

	ex.queue = make(map[string]*Writer)
	t0 := time.Now()
	var n uint

	err := Each(ex.Connection, ex.BatchSize, func(ns *Nameserver) (e error) {
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
		if n%ex.BatchSize == 0 {
			rate := perSec(n, time.Since(t0))
			title := fmt.Sprintf(titleFmt, os.Args[0], rate, ns.ID)
			gspt.SetProcTitle(title)
		}
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

func (ex *Exporter) exportEntry(channel string, ns *Nameserver) (err error) {
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

func perSec(n uint, t time.Duration) int {
	res := (time.Duration(n) * time.Second) / t
	return int(res)
}
