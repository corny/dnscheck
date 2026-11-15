package export

import (
	"encoding/csv"

	"github.com/corny/dnscheck/internal/store"
)

type csvExporter struct {
	*nsExporter

	writer *csv.Writer
}

var _ insExporter = &csvExporter{} // type check

func newCSVExporter(tpl string) (insExporter, error) {
	e := &csvExporter{
		nsExporter: &nsExporter{tpl: tpl, ext: "csv"},
	}

	if err := e.open(); err != nil {
		return nil, err
	}

	e.writer = csv.NewWriter(e.buf)
	if err := e.writer.Write(nameserverFields); err != nil {
		e.abort()
		return nil, err
	}

	return e, nil
}

func (e *csvExporter) write(ns *store.Nameserver) error {
	fields := make([]string, len(nameserverFields))

	for i, field := range nameserverFields {
		fields[i] = fieldString(ns, field)
	}

	return e.writer.Write(fields)
}

func (e *csvExporter) close() error {
	e.writer.Flush()
	if err := e.writer.Error(); err != nil {
		return err
	}
	return e.nsExporter.close()
}
