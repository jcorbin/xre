package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"syscall"
)

type command interface {
	Process(buf []byte, ateof bool) (off int, err error)
}

//// running commands

type filelike interface {
	Name() string
	Stat() (os.FileInfo, error)
	Fd() uintptr
}

func runCommand(cmd command, r io.Reader, useMmap bool) error {
	if f, canMmap := r.(filelike); useMmap && canMmap {
		buf, fin, err := mmap(f)
		if err == nil {
			defer fin()
			_, err = cmd.Process(buf, true)
		}
		return err
	}

	if rf, canReadFrom := cmd.(io.ReaderFrom); canReadFrom {
		_, err := rf.ReadFrom(r)
		return err
	}

	// TODO if (some) commands implement io.Writer, then could upgrade to r.(WriterTo)

	rb := readBuf{buf: make([]byte, 0, minRead)} // TODO configurable buffer size
	return rb.Process(cmd, r)
}

func mmap(f filelike) ([]byte, func() error, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: cannot stat %q: %v", f.Name(), err)
	}

	size := fi.Size()
	if size <= 0 {
		return nil, nil, fmt.Errorf("mmap: file %q has negative size", f.Name())
	}
	if size != int64(int(size)) {
		return nil, nil, fmt.Errorf("mmap: file %q is too large", f.Name())
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return data, func() error {
		return syscall.Munmap(data)
	}, nil
}

//// record addressing and selection

// TODO implement ReadFrom on every address selector, so that if it gets used
// as a top-level, we can default to scanning by some default split (e.g. lineSplitter(1))

type addrRange struct {
	start, end, n int
	next          command
}

func (ar *addrRange) Process(buf []byte, ateof bool) (off int, err error) {
	ar.n++
	if ar.start <= ar.n && ar.n <= ar.end {
		_, err = ar.next.Process(buf, ar.n == ar.end)
	}
	if ateof {
		ar.n = 0
	}
	return len(buf), err
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
		for li := 2; err == nil; {
			m := buf[off+locs[li] : off+locs[li+1]] // extracted sub-match
			li += 2
			if li < len(locs) {
				_, err = ex.next.Process(m, false)
			} else {
				_, err = ex.next.Process(m, true)
				break
			}
		}
		off += locs[1]
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

var errTooManyEmpties = errors.New("too many empty tokens without progressing")

func (bd betweenDelimSplit) Process(buf []byte, ateof bool) (off int, err error) {
	const maxConsecutiveEmptyReads = 100
	empties := 0
	for err == nil && off < len(buf) {
		var advance int
		var token []byte
		if advance, token, err = bd.split.Split(buf[off:], ateof); advance < 0 {
			if err == nil {
				err = bufio.ErrNegativeAdvance
			}
		} else if advance > len(buf)-off {
			if err == nil {
				err = bufio.ErrAdvanceTooFar
			}
		} else {
			off += advance
		}
		if err != nil || token == nil {
			if err == bufio.ErrFinalToken {
				_, err = bd.next.Process(token, true)
			}
			break
		}
		if advance > 0 {
			empties = 0
		} else {
			// Returning tokens not advancing input at EOF.
			if empties++; empties > maxConsecutiveEmptyReads {
				return off, errTooManyEmpties
			}
		}
		_, err = bd.next.Process(token, true)
	}
	return off, err
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

type accum struct {
	tmp  bytes.Buffer
	next command
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

func (ac *accum) Process(buf []byte, ateof bool) (off int, err error) {
	if buf != nil {
		_, _ = ac.tmp.Write(buf)
	}
	if ateof {
		_, err = ac.next.Process(ac.tmp.Bytes(), true) // TODO hack; reconsider ateof passing
		ac.tmp.Reset()
	}
	return len(buf), err
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
