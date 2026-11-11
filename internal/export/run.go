package export

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/corny/dnscheck/internal/store"
)

// Exporter writes all nameservers into files: nameservers-all contains all
// nameservers, nameservers the valid ones and <country code> the valid ones
// of each country.
type Exporter struct {
	Destination string // Destination is the output directory
	Store       *store.Store
	BatchSize   uint
}

// Run exports all nameservers. The existing files are only replaced if the
// export succeeds.
func (ex *Exporter) Run() error {
	if ex.BatchSize < 1 {
		return errors.New("batch size too small")
	}

	if ex.Store == nil {
		return errors.New("store missing")
	}

	if _, err := os.Stat(ex.Destination); err != nil {
		return fmt.Errorf("invalid destination %q: %w", ex.Destination, err)
	}

	writers := make(map[string]*Writer)
	t0 := time.Now()
	var n uint

	err := ex.Store.EachNameserver(ex.BatchSize, func(ns *store.Nameserver) error {
		if err := ex.write(writers, "nameservers-all", ns); err != nil {
			return err
		}
		if ns.IsValid() {
			if err := ex.write(writers, "nameservers", ns); err != nil {
				return err
			}
			if ns.Country != nil && *ns.Country != "" {
				if err := ex.write(writers, strings.ToLower(*ns.Country), ns); err != nil {
					return err
				}
			}
		}
		n++
		return nil
	})
	if err != nil {
		for _, w := range writers {
			w.Abort()
		}
		return err
	}

	var errs []error
	for name, w := range writers {
		if err := w.Close(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		if strings.HasPrefix(name, "nameservers") {
			slog.Info("exported", "file", name, "count", w.Count())
		} else {
			slog.Debug("exported", "file", name, "count", w.Count())
		}
	}
	slog.Info("export finished", "total", n, "duration", time.Since(t0))

	return errors.Join(errs...)
}

// write writes the nameserver into the files of the given name.
func (ex *Exporter) write(writers map[string]*Writer, name string, ns *store.Nameserver) error {
	w, found := writers[name]
	if !found {
		var err error
		w, err = NewWriter(ex.Destination, name)
		if err != nil {
			return err
		}
		writers[name] = w
	}

	return w.Write(ns)
}
