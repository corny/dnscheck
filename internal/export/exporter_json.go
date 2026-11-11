package export

import (
	"encoding/json"

	"github.com/corny/dnscheck/internal/store"
)

type jsonExporter struct {
	*nsExporter
}

var ( // type checks
	_ insExporter  = &jsonExporter{}
	_ nsSerializer = &jsonExporter{}
)

func newJSONExporter(tpl string) (insExporter, error) {
	e := &jsonExporter{nsExporter: &nsExporter{tpl: tpl}}
	if err := e.setup("json", "[", ",", "]", e); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *jsonExporter) convertNS(ns *store.Nameserver) ([]byte, error) {
	return json.Marshal(ns)
}
