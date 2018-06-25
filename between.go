package main

import (
	"bufio"
	"bytes"
	"io"
	"regexp"
)

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
