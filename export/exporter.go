package export

import (
	"bufio"
	"fmt"
	"os"
)

type nsSerializer interface {
	convertNS(*Nameserver) ([]byte, error)
}

type insExporter interface {
	write(*Nameserver) error
	close() error
}

type nsExporter struct {
	tpl        string
	ext        string // target file extension
	fileHandle *os.File
	buf        *bufio.Writer

	convert func(*Nameserver) ([]byte, error)

	prefix string // data written *before* the first record
	infix  string // data written *between* two records
	suffix string // data written *after* the last record

	hasPrefix         bool
	hasRecordsWritten bool
}

func (e *nsExporter) setup(ext, prefix, infix, suffix string, conv nsSerializer) (err error) {
	e.ext = ext
	e.prefix = prefix
	e.infix = infix
	e.suffix = suffix
	e.convert = conv.convertNS

	err = e.open()
	return
}

func (e *nsExporter) tmpName() string {
	return fmt.Sprintf("%s.%s.tmp", e.tpl, e.ext)
}

func (e *nsExporter) targetName() string {
	return fmt.Sprintf("%s.%s", e.tpl, e.ext)
}

func (e *nsExporter) prepareFile() (*os.File, error) {
	fo, err := os.OpenFile(e.tmpName(), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return fo, nil
}

func (e *nsExporter) open() error {
	f, err := e.prepareFile()
	if err != nil {
		return err
	}

	e.fileHandle = f
	e.buf = bufio.NewWriter(f)
	return nil
}

func (e *nsExporter) write(ns *Nameserver) (err error) {
	if !e.hasPrefix {
		e.hasPrefix = true
		_, err = e.buf.WriteString(e.prefix)
		if err != nil {
			return
		}
	}

	data, err := e.convert(ns)
	if err != nil {
		return
	}

	if len(data) > 0 {
		if e.hasRecordsWritten && e.infix != "" {
			_, err = e.buf.WriteString(e.infix)
			if err != nil {
				return
			}
		}
		_, err = e.buf.Write(data)
		if err != nil {
			return
		}
		e.hasRecordsWritten = true
	}

	return nil
}

func (e *nsExporter) close() error {
	if e.suffix != "" {
		e.buf.WriteString(e.suffix)
	}
	if err := e.buf.Flush(); err != nil {
		return err
	}
	if err := e.fileHandle.Close(); err != nil {
		return err
	}
	if err := os.Rename(e.tmpName(), e.targetName()); err != nil {
		return err
	}
	return nil
}
