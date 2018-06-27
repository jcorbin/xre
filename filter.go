package main

import (
	"errors"
	"fmt"
	"regexp"
)

func scanG(s string) (command, string, error) {
	var g gcommand
	var err error
	g.pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return g, s, nil
}

func scanV(s string) (command, string, error) {
	v := gcommand{negate: true}
	var err error
	v.pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return v, s, nil
}

type gcommand struct {
	negate bool
	pat    *regexp.Regexp
}

func (g gcommand) Create(nc command, env environment) (processor, error) {
	if g.pat == nil {
		if g.negate {
			return nil, errors.New("empty v command")
		}
		return nil, errors.New("empty g command")
	}

	next, err := create(nc, env)
	if err != nil {
		return nil, err
	}

	if g.negate {
		return filterNeg{g.pat, next}, nil
	}
	return filter{g.pat, next}, nil
}

type filter struct {
	pat  *regexp.Regexp
	next processor
}

type filterNeg struct {
	pat  *regexp.Regexp
	next processor
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

func (g gcommand) String() string {
	if g.negate {
		return fmt.Sprintf("v%v", regexpString(g.pat))
	}
	return fmt.Sprintf("g%v", regexpString(g.pat))
}
func (fl filter) String() string {
	return fmt.Sprintf("g%v%v", regexpString(fl.pat), fl.next)
}
func (fn filterNeg) String() string {
	return fmt.Sprintf("v%v%v", regexpString(fn.pat), fn.next)
}
