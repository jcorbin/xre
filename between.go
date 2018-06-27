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

func scanY(s string) (command, string, error) {
	if len(s) == 0 {
		// TODO could default to line-delimiting (aka as if y"\n" was given)
		return nil, s, fmt.Errorf("empty y command")
	}
	var y between
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		s = s[1:]
		y.open = c
		y.close = balancedOpens[c]

	case '/':
		var err error
		y.pat, s, err = scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}

	case '"':
		var err error
		y.delim, s, err = scanString(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		if len(s) > 3 && s[0] == '~' && s[1] == '"' {
			y.cutset, s, err = scanString(c, s[1:])
			if err != nil {
				return nil, s, err
			}
		}

	default:
		return nil, s, fmt.Errorf("unrecognized y command")
	}
	return y, s, nil
}

type between struct {
	// TODO support and optimize to static start/end byte strings when possible
	pat           *regexp.Regexp
	delim, cutset string
	open, close   byte
}

func (y between) Create(nc command, env environment) (processor, error) {
	if y.open == 0 && y.pat == nil && y.delim == "" {
		return nil, errors.New("empty y command")
	}

	next, err := create(nc, env)
	if err != nil {
		return nil, err
	}

	if y.open != 0 {
		return betweenBalanced{y.open, y.close, next}, nil
	}

	if y.pat != nil {
		return betweenDelimRe{y.pat, next}, nil
	}

	bds := betweenDelimSplit{next: next}
	if allNewlines(y.delim) && y.cutset == "" {
		bds.split = lineSplitter(len(y.delim))
	} else if len(y.delim) == 1 {
		bds.split = byteSplitter(y.delim[0])
	} else {
		bds.split = bytesSplitter(y.delim)
	}
	if y.cutset != "" {
		bds.split = trimmedSplitter(bds.split, y.cutset)
	}
	return bds, nil
}

type betweenBalanced struct {
	open, close byte
	next        processor
}

type betweenDelimRe struct {
	pat  *regexp.Regexp
	next processor
}

type betweenDelimSplit struct {
	split splitter
	next  processor
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

func (y between) String() string {
	if y.pat != nil {
		return fmt.Sprintf("y%v", regexpString(y.pat))
	}
	if y.delim != "" {
		if y.cutset != "" {
			return fmt.Sprintf(`y%q~%q`, y.delim, y.cutset)
		}
		return fmt.Sprintf(`y%q`, y.delim)
	}
	if y.open != 0 {
		return fmt.Sprintf("y%s", string(y.open))
	}
	return "y"
}
func (bb betweenBalanced) String() string {
	return fmt.Sprintf("y%s%v", string(bb.open), bb.next)
}
func (bd betweenDelimRe) String() string {
	return fmt.Sprintf("y%v%v", regexpString(bd.pat), bd.next)
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
