package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

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

//// parsing

func pLinker(format string, delim []byte) (linker, error) {
	return func(next command) (command, error) {
		if format != "" && delim != nil {
			format, delim = fmt.Sprintf("%s%s", format, delim), nil
		}
		if format == "" && len(delim) == 0 {
			return next, nil
		}
		// from here on we have either have format or delim

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
	}, nil
}

func scanP(s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case '%':
		if len(s) < 3 || s[1] != '"' {
			return nil, s, errors.New("missing format scring to p%")
		}
		s = s[1:]
		var format string
		format, s, err = scanString(s[0], s[1:])
		if err == nil {
			lnk, err = pLinker(format, nil)
		}

	case '"':
		var tmp string
		tmp, s, err = scanString(s[0], s[1:])
		if err == nil {
			lnk, err = pLinker("", []byte(tmp))
		}

	default:
		lnk, err = pLinker("", nil)
	}
	return lnk, s, err
}

func (fr fmter) String() string       { return fmt.Sprintf("p%%%q%v", fr.fmt, fr.next) }
func (dr delimer) String() string     { return fmt.Sprintf("p%q%v", dr.delim, dr.next) }
func (wr writer) String() string      { return "p" }
func (fw fmtWriter) String() string   { return fmt.Sprintf("p%%%q", fw.fmt) }
func (dw delimWriter) String() string { return fmt.Sprintf("p%q", dw.delim) }
