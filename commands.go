package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
)

type command interface {
	Process(buf []byte, ateof bool) (off int, err error)
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

func (ex extract) Process(buf []byte, ateof bool) (off int, err error) {
	for err == nil && off < len(buf) {
		loc := ex.pat.FindIndex(buf[off:])
		if loc == nil {
			break
		}
		m := buf[off+loc[0] : off+loc[1]] // extracted match
		if off += loc[1]; off < len(buf) {
			_, err = ex.next.Process(m, false)
		} else {
			_, err = ex.next.Process(m, ateof)
		}
	}
	return off, err
}

func (ex extractSub) Process(buf []byte, ateof bool) (off int, err error) {
	for err == nil && off < len(buf) {
		locs := ex.pat.FindSubmatchIndex(buf[off:])
		if locs == nil {
			break
		}
		m := buf[off+locs[2] : off+locs[3]] // extracted match
		if off += locs[1]; off < len(buf) {
			_, err = ex.next.Process(m, false)
		} else {
			_, err = ex.next.Process(m, ateof)
		}
	}
	return off, err
}

func (eb extractBalanced) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO escaping? quoting?
	level, start := 0, 0
	for ; err == nil && off < len(buf); off++ {
		switch buf[off] {
		case eb.open:
			if level == 0 {
				start = off + 1
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start:off] // extracted match
				_, err = eb.next.Process(m, false)
			}
		}
	}
	return off, err
}

func (eb extractBalancedInc) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO escaping? quoting?
	level, start := 0, 0
	for ; err == nil && off < len(buf); off++ {
		switch buf[off] {
		case eb.open:
			if level == 0 {
				start = off
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start : off+1] // extracted match
				_, err = eb.next.Process(m, false)
			}
		}
	}
	return off, err
}

//// extraction between patterns

type between struct {
	start, end *regexp.Regexp
	next       command
}

type betweenDelimRe struct {
	pat  *regexp.Regexp
	next command
}

type betweenDelimSplit struct {
	split splitter
	next  command
}

type splitter interface {
	Split(data []byte, atEOF bool) (advance int, token []byte, err error)
}

func (by between) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO inclusive variant?
	for err == nil && off < len(buf) {
		// find start pattern
		loc := by.start.FindIndex(buf[off:])
		if loc == nil {
			break
		}
		if off += loc[1]; off >= len(buf) {
			break
		}
		m := buf[off:] // start extracted match after match of start pattern

		// find end pattern
		loc = by.end.FindIndex(m)
		if loc == nil {
			break
		}
		off += loc[1]
		m = m[:loc[0]] // end extracted match before match of end pattern

		_, err = by.next.Process(m, false)
	}
	return off, err
}

func (bd betweenDelimRe) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO inclusive variant?
	for err == nil && off < len(buf) {
		loc := bd.pat.FindIndex(buf[off:])
		if loc == nil {
			if ateof {
				nextOff, err := bd.next.Process(buf[off:], true)
				return off + nextOff, err
			}
			break
		}
		m := buf[off : off+loc[0]] // extracted match
		off += loc[1]
		_, err = bd.next.Process(m, false)
	}
	return off, err
}

func (bd betweenDelimSplit) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO technically we should use a split func that doesn't consume the last partial if !ateof
	_, err = bd.ReadFrom(bytes.NewReader(buf))
	return len(buf), err // FIXME would be great to get the truth from ReadFrom below
}

func (bd betweenDelimSplit) ReadFrom(r io.Reader) (n int64, err error) {
	// TODO inclusive variant?
	sc := bufio.NewScanner(r)
	// sc.Buffer() // TODO raise the roof
	sc.Split(bd.split.Split)
	for err == nil && sc.Scan() {
		_, err = bd.next.Process(sc.Bytes(), false)
	}
	if scerr := sc.Err(); err == nil {
		err = scerr
	}
	// FIXME n is always 0, since there's no telling how many bytes the scanner consumed
	return n, err
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

func (fl filter) Process(buf []byte, ateof bool) (off int, err error) {
	if fl.pat.Match(buf) {
		return fl.next.Process(buf, ateof)
	}
	return 0, nil
}

func (fn filterNeg) Process(buf []byte, ateof bool) (off int, err error) {
	if !fn.pat.Match(buf) {
		return fn.next.Process(buf, ateof)
	}
	return 0, nil
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
