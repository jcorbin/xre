package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var errTooManyEmpties = errors.New("too many empty tokens without progressing")

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
	next, err := createProcessor(nc, env)
	if err != nil {
		return nil, err
	}

	if y.open != 0 {
		return betweenBalanced{y.open, y.close, next}, nil
	}
	if y.pat != nil {
		return betweenDelimRe{y.pat, next}, nil
	}
	m, err := y.createMatcher(env)
	if err != nil {
		return nil, err
	}
	return createMatcherCommand(m, next, env)
}

func (y between) createMatcher(env environment) (matcher, error) {
	if y.delim == "" {
		return nil, errors.New("empty y command")
	}
	var split splitter
	if allNewlines(y.delim) && y.cutset == "" {
		split = lineSplitter(len(y.delim))
	} else if len(y.delim) == 1 {
		split = byteSplitter(y.delim[0])
	} else {
		split = bytesSplitter(y.delim)
	}
	if y.cutset != "" {
		split = trimmedSplitter(split, y.cutset)
	}
	return betweenDelimSplit{split}, nil
}

type betweenDelimRe struct {
	pat  *regexp.Regexp
	next processor
}

type betweenDelimSplit struct{ split splitter }

type splitter interface {
	Split(data []byte, atEOF bool) (advance int, token []byte, err error)
}

func (bdr betweenDelimRe) Process(buf []byte, last bool) error {
	// TODO inclusive variant?
	for off := 0; off < len(buf); {
		loc := bdr.pat.FindIndex(buf[off:])
		if loc == nil {
			return bdr.next.Process(buf[off:], true)
		}
		m := buf[off : off+loc[0]] // extracted match
		off += loc[1]
		if err := bdr.next.Process(m, false); err != nil {
			return err
		}
	}
	return nil
}

func (bds betweenDelimSplit) match(mp *matchProcessor, buf []byte) error {
	// TODO refactor splitter; unify with matcher

	advance, token, err := bds.split.Split(buf, mp.buf.Err() == io.EOF)
	if err == nil {
		if advance < 0 {
			err = bufio.ErrNegativeAdvance
		} else if advance > len(buf) {
			err = bufio.ErrAdvanceTooFar
		}
	}
	if err != nil || token == nil {
		return err
	}

	// XXX hack to extract the offset of token in buf without resorting
	// to pointer math, or modifying the split contract; fix in splitter
	// unification
	start := cap(buf) - cap(token)
	end := start + len(token)
	return mp.pushLoc(start, end, advance)
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

func (bb betweenBalanced) String() string { return fmt.Sprintf("y%s%v", string(bb.open), bb.next) }
func (bdr betweenDelimRe) String() string {
	return fmt.Sprintf("y%v%v", regexpString(bdr.pat), bdr.next)
}
func (bds betweenDelimSplit) String() string { return fmt.Sprintf("y%v", bds.split) }

func (ls lineSplitter) String() string        { return fmt.Sprintf("%q", strings.Repeat("\n", int(ls))) }
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
