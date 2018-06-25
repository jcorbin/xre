package main

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var errTooManyEmpties = errors.New("too many empty tokens without progressing")

var balancedOpens = map[byte]byte{
	'[': ']',
	'{': '}',
	'(': ')',
	'<': '>',
}

func scanY(s string) (linker, string, error) {
	if len(s) == 0 {
		// TODO could default to line-delimiting (aka as if y"\n" was given)
		return nil, s, fmt.Errorf("empty y command")
	}
	var y linker
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		s = s[1:]
		y = yBalLinker(c, balancedOpens[c])

	case '/':
		// TODO disabled y/start/end/ for now due to parsing ambiguity
		var start, end *regexp.Regexp
		var err error
		start, s, err = scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		// if len(s) > 0 {
		// 	end, s, err = scanPat(c, s)
		// 	if err != nil {
		// 		return nil, s, err
		// 	}
		// }
		y = yReLinker(start, end)

	case '"':
		var delim, cutset string
		var err error
		delim, s, err = scanString(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		if len(s) > 3 && s[0] == '~' && s[1] == '"' {
			cutset, s, err = scanString(c, s[1:])
			if err != nil {
				return nil, s, err
			}
		}
		y = yDelimLinker(delim, cutset)

	default:
		return nil, s, fmt.Errorf("unrecognized y command")
	}
	return y, s, nil
}

func yBalLinker(start, end byte) linker {
	return func(next command) (command, error) {
		return betweenBalanced{start, end, next}, nil
	}
}

func yReLinker(start, end *regexp.Regexp) linker {
	return func(next command) (command, error) {
		if end != nil {
			return betweenRe{start, end, next}, nil
		}
		return betweenDelimRe{start, next}, nil
	}
}

func yDelimLinker(delim, cutset string) linker {
	return func(next command) (command, error) {
		if len(delim) == 0 {
			return nil, errors.New("empty y\"delimiter\"")
		}
		bds := betweenDelimSplit{next: next}
		if allNewlines(delim) {
			bds.split = lineSplitter(len(delim))
		} else if len(delim) == 1 {
			bds.split = byteSplitter(delim[0])
		} else {
			bds.split = bytesSplitter(delim)
		}
		if cutset != "" {
			bds.split = trimmedSplitter(bds.split, cutset)
		}
		return bds, nil
	}
}

type betweenBalanced struct {
	open, close byte
	next        command
}

type betweenRe struct {
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

func (bb betweenBalanced) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO escaping? quoting?
	level, start := 0, 0
	for ; err == nil && off < len(buf); off++ {
		switch buf[off] {
		case bb.open:
			if level == 0 {
				start = off + 1
			}
			level++
		case bb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start:off] // extracted match
				_, err = bb.next.Process(m, false)
			}
		}
	}
	return off, err
}

func (by betweenRe) Process(buf []byte, ateof bool) (off int, err error) {
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

func (bds betweenDelimSplit) Process(buf []byte, ateof bool) (off int, err error) {
	const maxConsecutiveEmptyReads = 100
	empties := 0
	for err == nil && off < len(buf) {
		var advance int
		var token []byte
		if advance, token, err = bds.split.Split(buf[off:], ateof); advance < 0 {
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
				_, err = bds.next.Process(token, true)
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
		_, err = bds.next.Process(token, true)
	}
	return off, err
}

func (bb betweenBalanced) String() string {
	return fmt.Sprintf("y%s%v", string(bb.open), bb.next)
}
func (by betweenRe) String() string {
	return fmt.Sprintf("y/%v/%v/%v", regexpString(by.start), regexpString(by.end), by.next)
}
func (bd betweenDelimRe) String() string {
	return fmt.Sprintf("y/%v/%v", regexpString(bd.pat), bd.next)
}
func (bds betweenDelimSplit) String() string {
	return fmt.Sprintf("y%v%v", bds.split, bds.next)
}

func (ls lineSplitter) String() string        { return fmt.Sprintf("%q", strings.Repeat(`\n`, int(ls))) }
func (bs byteSplitter) String() string        { return fmt.Sprintf("%q", string(bs)) }
func (bss bytesSplitter) String() string      { return fmt.Sprintf("%q", []byte(bss)) }
func (bst byteSplitTrimmer) String() string   { return fmt.Sprintf("%q~%q", bst.delim, bst.cutset) }
func (bsst bytesSplitTrimmer) String() string { return fmt.Sprintf("%q~%q", bsst.delim, bsst.cutset) }

func allNewlines(delim string) bool {
	for i := 0; i < len(delim); i++ {
		if delim[i] != '\n' {
			return false
		}
	}
	return true
}
