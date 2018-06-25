package main

import (
	"bufio"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var errTooManyEmpties = errors.New("too many empty tokens without progressing")

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

//// parsing

func yReLinker(start, end *regexp.Regexp) (linker, error) {
	return func(next command) (command, error) {
		if end != nil {
			return between{start, end, next}, nil
		}
		return betweenDelimRe{start, next}, nil
	}, nil
}

func yDelimLinker(delim, cutset string) (linker, error) {
	return func(next command) (command, error) {
		if len(delim) == 0 {
			return nil, errors.New("empty y\"delimiter\"")
		}
		if allNewlines(delim) {
			return betweenDelimSplit{lineSplitter(len(delim)), next}, nil
		}
		if cutset != "" {
			if len(delim) == 1 {
				return betweenDelimSplit{byteSplitTrimmer{delim[0], cutset}, next}, nil
			}
			return betweenDelimSplit{bytesSplitTrimmer{[]byte(delim), cutset}, next}, nil
		}
		if len(delim) == 1 {
			return betweenDelimSplit{byteSplitter(delim[0]), next}, nil
		}
		return betweenDelimSplit{bytesSplitter(delim), next}, nil
	}, nil
}

func allNewlines(delim string) bool {
	for i := 0; i < len(delim); i++ {
		if delim[i] != '\n' {
			return false
		}
	}
	return true
}

var balancedOpens = map[byte]byte{
	'[': ']',
	'{': '}',
	'(': ')',
	'<': '>',
}

func scanY(s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case '[', '{', '(', '<':
		s = s[1:]
		lnk, err = xBalLinker(c, balancedOpens[c], false)

	case '/':
		s = s[1:]
		var pats [2]*regexp.Regexp
		for i := 0; len(s) > 0 && i < 2; i++ {
			pats[i], s, err = scanPat(c, s)
			if err != nil {
				break
			}
		}
		if err == nil {
			lnk, err = yReLinker(pats[0], pats[1])
		}

	case '"':
		var delim, cutset string
		delim, s, err = scanString(c, s[1:])
		if err == nil {
			if len(s) > 3 && s[0] == '~' && s[1] == '"' {
				cutset, s, err = scanString(c, s[1:])
			}
		}
		if err == nil {
			lnk, err = yDelimLinker(delim, cutset)
		}

	default:
		// TODO could default to line-delimiting (aka as if y"\n" was given)
		err = fmt.Errorf("unrecognized y command")
	}
	return lnk, s, err
}

func (by between) String() string {
	return fmt.Sprintf("y/%v/%v/%v", regexpString(by.start), regexpString(by.end), by.next)
}
func (bd betweenDelimRe) String() string {
	return fmt.Sprintf("y/%v/%v", regexpString(bd.pat), bd.next)
}
func (bd betweenDelimSplit) String() string {
	return fmt.Sprintf("y%v%v", bd.split, bd.next)
}

func (ls lineSplitter) String() string        { return fmt.Sprintf("%q", strings.Repeat(`\n`, int(ls))) }
func (bs byteSplitter) String() string        { return fmt.Sprintf("%q", string(bs)) }
func (bss bytesSplitter) String() string      { return fmt.Sprintf("%q", []byte(bss)) }
func (bst byteSplitTrimmer) String() string   { return fmt.Sprintf("%q~%q", bst.delim, bst.cutset) }
func (bsst bytesSplitTrimmer) String() string { return fmt.Sprintf("%q~%q", bsst.delim, bsst.cutset) }
