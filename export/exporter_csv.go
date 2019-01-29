package export

import "encoding/csv"

type csvExporter struct {
	*nsExporter

	writer *csv.Writer
}

var (
	// type checks
	_ insExporter  = &csvExporter{}
	_ nsSerializer = &csvExporter{}
)

func newCSVExporter(tpl string) (insExporter, error) {
	e := &csvExporter{
		nsExporter: &nsExporter{tpl: tpl, ext: "csv"},
	}

	if err := e.open(); err != nil {
		return nil, err
	}

	e.writer = csv.NewWriter(e.fileHandle)
	e.writer.Write(nameserverFields)

	return e, nil
}

// with code from encoding/csv (https://golang.org/src/encoding/csv/writer.go#L40)
// but without options (Comma, UseCLRF)
func (e *csvExporter) write(ns *Nameserver) error {
	fields := make([]string, len(nameserverFields))

	for i, field := range nameserverFields {
		fields[i] = ns.GetString(field)
	}

	return e.writer.Write(fields)
}

func (e *csvExporter) convertNS(_ *Nameserver) ([]byte, error) {
	return nil, nil
}
