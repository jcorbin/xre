package main

import (
	"fmt"
	"regexp"
)

func scanG(s string) (linker, string, error) {
	var pat *regexp.Regexp
	var err error
	pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	g := gLinker(pat)
	return g, s, nil
}

func scanV(s string) (linker, string, error) {
	var pat *regexp.Regexp
	var err error
	pat, s, err = scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	v := vLinker(pat)
	return v, s, nil
}

func gLinker(pat *regexp.Regexp) linker {
	return func(next command) (command, error) {
		return filter{pat, next}, nil
	}
}

func vLinker(pat *regexp.Regexp) linker {
	return func(next command) (command, error) {
		return filterNeg{pat, next}, nil
	}
}

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

func (fl filter) String() string {
	return fmt.Sprintf("g/%v/%v", regexpString(fl.pat), fl.next)
}
func (fn filterNeg) String() string {
	return fmt.Sprintf("v/%v/%v", regexpString(fn.pat), fn.next)
}
