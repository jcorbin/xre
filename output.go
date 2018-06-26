package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func scanP(s string) (command, string, error) {
	var p print
	if len(s) == 0 {
		return p, s, nil
	}
	switch c := s[0]; c {

	case '%':
		if len(s) < 3 || s[1] != '"' {
			return nil, s, errors.New("missing format string to p%")
		}
		var err error
		p.fmt, s, err = scanString(s[1], s[2:])
		if err != nil {
			return nil, s, err
		}

	case '"':
		var err error
		p.delim, s, err = scanString(s[0], s[1:])
		if err != nil {
			return nil, s, err
		}

	default:
		return nil, s, fmt.Errorf("unrecognized p command")
	}
	return p, s, nil
}

type print struct {
	fmt, delim string
	// TODO destination control
}

func (p print) Create(nc command, env environment) (processor, error) {
	next, err := create(nc, env)
	if err != nil {
		return nil, err
	}

	if p.fmt == "" && p.delim == "" {
		return next, nil
	}

	// have either p.fmt or delim
	var delim []byte
	if p.delim != "" {
		if p.fmt != "" {
			p.fmt += p.delim
		} else {
			delim = []byte(p.delim)
		}
	}

	switch impl := next.(type) {
	case writer:
		if p.fmt != "" {
			return fmtWriter{fmt: p.fmt, w: impl.w}, nil
		}
		return delimWriter{delim: delim, w: impl.w}, nil

	case fmtWriter:
		if p.fmt != "" {
			return fmtWriter{fmt: p.fmt + impl.fmt, w: impl.w}, nil
		}
		return delimWriter{delim: delim, w: impl.w}, nil

	case delimWriter:
		if p.fmt != "" {
			return fmtWriter{fmt: p.fmt + string(impl.delim), w: impl.w}, nil
		}
		return delimWriter{delim: append(delim, impl.delim...), w: impl.w}, nil

	default:
		if p.fmt != "" {
			return &fmter{fmt: p.fmt, next: next}, nil
		}
		return &delimer{delim: delim, next: next}, nil
	}
}

type fmter struct {
	fmt  string
	tmp  bytes.Buffer
	next processor
}

type delimer struct {
	delim []byte
	tmp   bytes.Buffer
	next  processor
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

func (p print) String() string {
	if p.fmt != "" {
		return fmt.Sprintf("p%%%q", p.fmt)
	}
	if p.delim != "" {
		return fmt.Sprintf("p%q", p.delim)
	}
	return "p"
}
func (fr fmter) String() string       { return fmt.Sprintf("p%%%q%v", fr.fmt, fr.next) }
func (dr delimer) String() string     { return fmt.Sprintf("p%q%v", dr.delim, dr.next) }
func (wr writer) String() string      { return "p" }
func (fw fmtWriter) String() string   { return fmt.Sprintf("p%%%q", fw.fmt) }
func (dw delimWriter) String() string { return fmt.Sprintf("p%q", dw.delim) }
