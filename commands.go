package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
)

type command interface {
	// Batch mode
	Process(buf []byte) error
}

type streamingCommand interface {
	command
	ProcessIn(buf []byte, last bool) (int, error)
}

//// extraction by pattern

type extract struct {
	pat  *regexp.Regexp
	next command
}

type extractSub struct {
	pat  *regexp.Regexp
	next command
}

type extractBalanced struct {
	open, close byte
	next        command
}

type extractBalancedInc struct {
	open, close byte
	next        command
}

func (ex extract) Process(buf []byte) error {
	for b := buf; len(b) > 0; {
		loc := ex.pat.FindIndex(b)
		if loc == nil {
			break
		}
		m := b[loc[0]:loc[1]] // extracted match
		if i := loc[1] + 1; i < len(b) {
			b = b[i:]
		} else {
			b = nil
		}
		if err := ex.next.Process(m); err != nil {
			return err
		}
	}
	return nil
}

func (ex extractSub) Process(buf []byte) error {
	for b := buf; len(b) > 0; {
		locs := ex.pat.FindSubmatchIndex(b)
		if locs == nil {
			break
		}
		m := b[locs[2]:locs[3]] // extracted match
		if i := locs[1] + 1; i < len(b) {
			b = b[i:]
		} else {
			b = nil
		}
		if err := ex.next.Process(m); err != nil {
			return err
		}
	}
	return nil
}

func (eb extractBalanced) Process(buf []byte) error {
	// TODO escaping? quoting?
	level := 0
	start, end := 0, 0
	for i := 0; i < len(buf); i++ {
		switch buf[i] {
		case eb.open:
			if level == 0 {
				start = i + 1
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				end = i
				m := buf[start:end] // extracted match
				if err := eb.next.Process(m); err != nil {
					return err
				}
				start, end = 0, 0
			}
		}
	}
	return nil
}

func (eb extractBalancedInc) Process(buf []byte) error {
	// TODO escaping? quoting?
	level := 0
	start, end := 0, 0
	for i := 0; i < len(buf); i++ {
		switch buf[i] {
		case eb.open:
			if level == 0 {
				start = i
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				end = i + 1
				m := buf[start:end] // extracted match
				if err := eb.next.Process(m); err != nil {
					return err
				}
				start, end = 0, 0
			}
		}
	}
	return nil
}

//// extraction between patterns

type between struct {
	start, end *regexp.Regexp
	next       command
}

type betweenDelim struct {
	pat  *regexp.Regexp
	next command
}

func (by between) Process(buf []byte) error {
	// TODO inclusive / exclusive control
	for b := buf; len(b) > 0; {
		// find start pattern
		loc := by.start.FindIndex(b)
		if loc == nil {
			break
		}
		m := b[loc[0]:] // extracted match (start)
		b = b[loc[1]+1:]

		// find end pattern
		off := loc[1] - loc[0]
		loc = by.end.FindIndex(b)
		if loc == nil {
			break
		}
		m = m[:off+loc[1]+1] // extracted match (end)
		b = b[loc[1]+1:]

		if err := by.next.Process(m); err != nil {
			return err
		}
	}
	return nil
}

func (bd betweenDelim) Process(buf []byte) error {
	// TODO inclusive / exclusive control
	b := buf
	for len(b) > 0 {
		loc := bd.pat.FindIndex(b)
		if loc == nil {
			break
		}
		i := loc[1]
		if i < len(b) {
			i++
		}
		m := b[:i] // extracted match
		b = b[i:]
		if err := bd.next.Process(m); err != nil {
			return err
		}
	}
	return bd.next.Process(b)
}

//// filtering

type filter struct {
	pat  *regexp.Regexp
	next command
}

type filterNeg struct {
	pat  *regexp.Regexp
	next command
}

func (fl filter) Process(buf []byte) error {
	if fl.pat.Match(buf) {
		return fl.next.Process(buf)
	}
	return nil
}

func (fn filterNeg) Process(buf []byte) error {
	if !fn.pat.Match(buf) {
		return fn.next.Process(buf)
	}
	return nil
}

//// formatting and output

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

func (fr fmter) Process(buf []byte) error {
	fr.tmp.Reset()
	_, err := fmt.Fprintf(&fr.tmp, fr.fmt, buf)
	if err == nil {
		err = fr.next.Process(fr.tmp.Bytes())
	}
	return err
}

func (dr delimer) Process(buf []byte) error {
	dr.tmp.Reset()
	_, err := dr.tmp.Write(buf)
	if err == nil {
		_, err = dr.tmp.Write(dr.delim)
	}
	if err == nil {
		err = dr.next.Process(dr.tmp.Bytes())
	}
	return err
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

func (wr writer) Process(buf []byte) error {
	_, err := wr.w.Write(buf)
	return err
}

func (fw fmtWriter) Process(buf []byte) error {
	_, err := fmt.Fprintf(fw.w, fw.fmt, buf)
	return err
}

func (dw delimWriter) Process(buf []byte) error {
	_, err := dw.w.Write(buf)
	if err == nil {
		_, err = dw.w.Write(dw.delim)
	}
	return err
}
