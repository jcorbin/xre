package main

import "regexp"

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
