package main

import (
	"errors"
	"fmt"
	"regexp"
)

func scanX(s string) (command, string, error) {
	if len(s) == 0 {
		return nil, s, fmt.Errorf("empty x command")
	}
	var x extract
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		s = s[1:]
		x.open = c
		x.close = balancedOpens[c]

	case '/':
		var err error
		x.pat, s, err = scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}

	default:
		return nil, s, fmt.Errorf("unrecognized x command")
	}
	return x, s, nil
}

type extract struct {
	pat         *regexp.Regexp
	open, close byte
}

func (x extract) Create(nc command, env environment) (processor, error) {
	next, err := createProcessor(nc, env)
	if err != nil {
		return nil, err
	}

	if x.open != 0 {
		return extractBalanced{x.open, x.close, next}, nil
	}
	m, err := x.createMatcher(env)
	if err != nil {
		return nil, err
	}
	return createMatcherCommand(m, next, env)
}

func (x extract) createMatcher(env environment) (matcher, error) {
	if x.pat == nil {
		return nil, errors.New("empty x command")
	}
	switch n := x.pat.NumSubexp(); n {
	case 0:
		return extractRe{x.pat}, nil
	case 1:
		return extractReSub{x.pat}, nil
	default:
		return nil, fmt.Errorf("extraction with %v sub-patterns not supported", n)
	}
}

type extractRe struct{ pat *regexp.Regexp }
type extractReSub extractRe

func (er extractRe) match(mp *matchProcessor, buf []byte) error {
	if loc := er.pat.FindIndex(buf); loc != nil {
		return mp.pushLoc(loc[0], loc[1], loc[1])
	}
	return nil
}

func (ers extractReSub) match(mp *matchProcessor, buf []byte) error {
	if locs := ers.pat.FindSubmatchIndex(buf); locs != nil {
		return mp.pushLoc(locs[2], locs[3], locs[1])
	}
	return nil
}

func (x extract) String() string {
	if x.pat != nil {
		return fmt.Sprintf("x%v", regexpString(x.pat))
	}
	if x.open != 0 {
		return fmt.Sprintf("x%s", string(x.open))
	}
	return "x"
}

func (eb extractBalanced) String() string { return fmt.Sprintf("x%s%v", string(eb.open), eb.next) }
func (er extractRe) String() string       { return fmt.Sprintf("x%v", regexpString(er.pat)) }
func (ers extractReSub) String() string   { return fmt.Sprintf("x%v", regexpString(ers.pat)) }
