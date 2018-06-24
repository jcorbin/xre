package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func scanP(s string) (linker, string, error) {
	var p linker
	if len(s) == 0 {
		p = pLinker("", "")
		return p, s, nil
	}
	switch c := s[0]; c {

	case '%':
		if len(s) < 3 || s[1] != '"' {
			return nil, s, errors.New("missing format string to p%")
		}
		var format string
		var err error
		format, s, err = scanString(s[1], s[2:])
		if err != nil {
			return nil, s, err
		}
		p = pLinker(format, "")

	case '"':
		var delim string
		var err error
		delim, s, err = scanString(s[0], s[1:])
		if err != nil {
			return nil, s, err
		}
		p = pLinker("", delim)

	default:
		return nil, s, fmt.Errorf("unrecognized p command")
	}
	return p, s, nil
}

func pLinker(format, sdelim string) linker {
	if format == "" && sdelim == "" {
		return func(next command) (command, error) {
			return next, nil
		}
	}

	// have either format or delim
	var delim []byte
	if sdelim != "" {
		if format != "" {
			format += sdelim
		} else {
			delim = []byte(sdelim)
		}
	}

	return func(next command) (command, error) {
		switch nc := next.(type) {
		case writer:
			if format != "" {
				return fmtWriter{fmt: format, w: nc.w}, nil
			}
			return delimWriter{delim: delim, w: nc.w}, nil

		case fmtWriter:
			if format != "" {
				return fmtWriter{fmt: format + nc.fmt, w: nc.w}, nil
			}
			return delimWriter{delim: delim, w: nc.w}, nil

		case delimWriter:
			if format != "" {
				return fmtWriter{fmt: format + string(nc.delim), w: nc.w}, nil
			}
			return delimWriter{delim: append(delim, nc.delim...), w: nc.w}, nil

		default:
			if format != "" {
				return &fmter{fmt: format, next: next}, nil
			}
			return &delimer{delim: delim, next: next}, nil
		}
	}
}

type fmter struct {
	fmt  string
	tmp  bytes.Buffer
	next command
}

type delimer struct {
	delim []byte
	tmp   bytes.Buffer
	next  command
}

type writer struct {
	w io.Writer
}

type fmtWriter struct {
	fmt string
	w   io.Writer
}

type delimWriter struct {
	delim []byte
	w     io.Writer
}

func (fr *fmter) Process(buf []byte, ateof bool) (off int, err error) {
	fr.tmp.Reset()
	_, _ = fmt.Fprintf(&fr.tmp, fr.fmt, buf)
	return fr.next.Process(fr.tmp.Bytes(), ateof)
}

func (dr *delimer) Process(buf []byte, ateof bool) (off int, err error) {
	dr.tmp.Reset()
	_, _ = dr.tmp.Write(buf)
	_, _ = dr.tmp.Write(dr.delim)
	return dr.next.Process(dr.tmp.Bytes(), ateof)
}

func (wr writer) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = wr.w.Write(buf)
	return len(buf), err
}

func (fw fmtWriter) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = fmt.Fprintf(fw.w, fw.fmt, buf)
	return len(buf), err
}

func (dw delimWriter) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = dw.w.Write(buf)
	if err == nil {
		_, err = dw.w.Write(dw.delim)
	}
	return len(buf), err
}

func (fr fmter) String() string       { return fmt.Sprintf("p%%%q%v", fr.fmt, fr.next) }
func (dr delimer) String() string     { return fmt.Sprintf("p%q%v", dr.delim, dr.next) }
func (wr writer) String() string      { return "p" }
func (fw fmtWriter) String() string   { return fmt.Sprintf("p%%%q", fw.fmt) }
func (dw delimWriter) String() string { return fmt.Sprintf("p%q", dw.delim) }
