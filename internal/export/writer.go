package export

import (
	"path"

	"github.com/corny/dnscheck/internal/store"
)

// A Writer writes nameservers into one file per format, namely
// <pathname>/<basename>.{txt,csv,json}. The files are written to temporary
// files first and renamed on Close.
type Writer struct {
	exporters []insExporter
	count     uint // number of records written
}

var knownExporters = []func(string) (insExporter, error){
	newTextExporter,
	newCSVExporter,
	newJSONExporter,
}

// NewWriter creates the temporary files for all formats.
func NewWriter(pathname, basename string) (*Writer, error) {
	w := &Writer{}
	tpl := path.Join(pathname, basename)

	for _, newExporter := range knownExporters {
		e, err := newExporter(tpl)
		if err != nil {
			w.Abort()
			return nil, err
		}
		w.exporters = append(w.exporters, e)
	}

	return w, nil
}

// Write writes a nameserver into all files.
func (w *Writer) Write(ns *store.Nameserver) error {
	for _, e := range w.exporters {
		if err := e.write(ns); err != nil {
			return err
		}
	}
	w.count++
	return nil
}

// Close finishes all files and moves them to their target names. On error,
// the remaining temporary files are removed.
func (w *Writer) Close() error {
	for i, e := range w.exporters {
		if err := e.close(); err != nil {
			for _, rest := range w.exporters[i:] {
				rest.abort()
			}
			return err
		}
	}
	return nil
}

// Abort closes and removes all temporary files.
func (w *Writer) Abort() {
	for _, e := range w.exporters {
		e.abort()
	}
}

// Count returns the number of records written
func (w *Writer) Count() uint {
	return w.count
}
