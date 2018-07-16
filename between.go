package xre

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

var errTooManyEmpties = errors.New("too many empty tokens without progressing")

func scanY(s string) (Command, string, error) {
	if len(s) == 0 {
		// TODO could default to line-delimiting (aka as if y"\n" was given)
		return nil, s, fmt.Errorf("empty y command")
	}
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		s = s[1:]
		return ProtoCommand{betweenBalanced{c, balancedOpens[c]}}, s, nil

	case '/':
		// TODO support and optimize to static byte strings when possible
		pat, s, err := scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		return ProtoCommand{betweenDelimRe{pat}}, s, nil

	case '"':
		delim, s, err := scanString(c, s[1:])
		var cutset string
		if delim == "" {
			return nil, s, fmt.Errorf("empty y command")
		}
		if err != nil {
			return nil, s, err
		}
		if len(s) > 3 && s[0] == '~' && s[1] == '"' {
			cutset, s, err = scanString(c, s[2:])
			if err != nil {
				return nil, s, err
			}
		}
		return ProtoCommand{betweenDelim(delim, cutset)}, s, nil

	default:
		return nil, s, fmt.Errorf("unrecognized y command")
	}
}

func betweenDelim(delim, cutset string) (bds betweenDelimSplit) {
	if allNewlines(delim) && cutset == "" {
		bds.split = lineSplitter(len(delim))
	} else if len(delim) == 1 {
		bds.split = byteSplitter(delim[0])
	} else {
		bds.split = bytesSplitter(delim)
	}
	if cutset != "" {
		bds.split = trimmedSplitter(bds.split, cutset)
	}
	return bds
}

type betweenDelimRe struct{ pat *regexp.Regexp }
type betweenDelimSplit struct{ split splitter }

type splitter interface {
	Split(data []byte, atEOF bool) (advance int, token []byte, err error)
}

func (bdr betweenDelimRe) match(mp *matchProcessor, buf []byte) error {
	if loc := bdr.pat.FindIndex(buf); loc != nil {
		return mp.pushLoc(0, loc[0], loc[1])
	}
	if mp.buf.Err() == io.EOF {
		return mp.flushTrailer()
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

func (bdr betweenDelimRe) Create(next Processor) Processor {
	return &matchProcessor{next: next, matcher: bdr}
}
func (bds betweenDelimSplit) Create(next Processor) Processor {
	return &matchProcessor{next: next, matcher: bds}
}

func (bb betweenBalanced) String() string    { return fmt.Sprintf("y%s", string(bb.open)) }
func (bdr betweenDelimRe) String() string    { return fmt.Sprintf("y%v", regexpString(bdr.pat)) }
func (bds betweenDelimSplit) String() string { return fmt.Sprintf("y%v", bds.split) }

func (ls lineSplitter) String() string   { return fmt.Sprintf("%q", strings.Repeat("\n", int(ls))) }
func (bs byteSplitter) String() string   { return fmt.Sprintf("%q", string(bs)) }
func (bss bytesSplitter) String() string { return fmt.Sprintf("%q", []byte(bss)) }
func (bst byteSplitTrimmer) String() string {
	return fmt.Sprintf("%q~%q", string(bst.delim), bst.cutset)
}
func (bsst bytesSplitTrimmer) String() string { return fmt.Sprintf("%q~%q", bsst.delim, bsst.cutset) }

func allNewlines(delim string) bool {
	for i := 0; i < len(delim); i++ {
		if delim[i] != '\n' {
			return false
		}
	}
	return true
}
