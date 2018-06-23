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

//// parsing

func scanG(s string) (linker, string, error) { return scanGV(false, s) }
func scanV(s string) (linker, string, error) { return scanGV(true, s) }
func scanGV(neg bool, s string) (lnk linker, _ string, _ error) {
	re, s, err := scanPat(s[0], s[1:])
	if err == nil {
		lnk, err = gLinker(re, neg)
	}
	return lnk, s, err
}
