package export

import "github.com/corny/dnscheck/internal/store"

type textExporter struct {
	*nsExporter
}

var ( // type checks
	_ insExporter  = &textExporter{}
	_ nsSerializer = &textExporter{}
)

func newTextExporter(tpl string) (insExporter, error) {
	e := &textExporter{nsExporter: &nsExporter{tpl: tpl}}
	if err := e.setup("txt", "", "\n", "", e); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *textExporter) convertNS(ns *store.Nameserver) ([]byte, error) {
	return []byte(ns.Address), nil
}
