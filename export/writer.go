package export

import "path"
import "sync"

// A Writer waites for new Nameserver instances to process and write away.
type Writer struct {
	Channel chan *Nameserver

	exporter []insExporter
	wg       sync.WaitGroup
	count    uint // number of records written
}

var knownExporters = []func(string) (insExporter, error){
	newTextExporter,
	newCSVExporter,
	newJSONExporter,
}

// NewWriter starts a new Writer life cycle. It returns a channel to which
// you send Nameserver instances. These are then transformed into different
// files (namely `${pathname}/${basename}.{txt,csv,json}`), so you need
// to pass in a path- and base name. When you're done, just close the
// done-channel and the internal caches are flushed and the files are
// closed.
func NewWriter(pathname, basename string) (*Writer, error) {
	w := &Writer{
		Channel:  make(chan *Nameserver, 100),
		exporter: make([]insExporter, len(knownExporters)),
	}

	tpl := path.Join(pathname, basename)
	for i, exFunc := range knownExporters {
		if ex, err := exFunc(tpl); err == nil {
			w.exporter[i] = ex
		}
	}

	w.wg.Add(1)
	go w.run()
	return w, nil
}

// Close finishes the current Write life cycle.
func (w *Writer) Close() error {
	close(w.Channel)
	w.wg.Wait()
	return nil
}

// Count returns the number of records written
func (w *Writer) Count() uint {
	return w.count
}

func (w *Writer) run() {
	for ns := range w.Channel {
		for _, e := range w.exporter {
			if err := e.write(ns); err != nil {
				panic(err)
			}
		}
		w.count++
	}

	for _, e := range w.exporter {
		if err := e.close(); err != nil {
			panic(err)
		}
	}

	w.wg.Done()
}
