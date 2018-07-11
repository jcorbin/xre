package main

import (
	"errors"
	"fmt"
	"regexp"
)

func scanG(s string) (command, string, error) {
	var g filter
	var err error
	g.pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return g, s, nil
}

func scanV(s string) (command, string, error) {
	v := filter{negate: true}
	var err error
	v.pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return v, s, nil
}

type filter struct {
	negate bool
	pat    *regexp.Regexp
}

func (g filter) Create(nc command, env environment) (processor, error) {
	if g.pat == nil {
		if g.negate {
			return nil, errors.New("empty v command")
		}
		return nil, errors.New("empty g command")
	}

	next, err := createProcessor(nc, env)
	if err != nil {
		return nil, err
	}

	if g.negate {
		return regexpNegFilter{g.pat, next}, nil
	}
	return regexpFilter{g.pat, next}, nil
}

type regexpFilter struct {
	pat  *regexp.Regexp
	next processor
}

type regexpNegFilter struct {
	pat  *regexp.Regexp
	next processor
}

func (fl regexpFilter) Process(buf []byte, last bool) error {
	if fl.pat.Match(buf) {
		return fl.next.Process(buf, last)
	}
	return nil
}

func (fn regexpNegFilter) Process(buf []byte, last bool) error {
	if !fn.pat.Match(buf) {
		return fn.next.Process(buf, last)
	}
	return nil
}

func (g filter) String() string {
	if g.negate {
		return fmt.Sprintf("v%v", regexpString(g.pat))
	}
	return fmt.Sprintf("g%v", regexpString(g.pat))
}
func (fl regexpFilter) String() string {
	return fmt.Sprintf("g%v%v", regexpString(fl.pat), fl.next)
}
func (fn regexpNegFilter) String() string {
	return fmt.Sprintf("v%v%v", regexpString(fn.pat), fn.next)
}
