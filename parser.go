package main

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var errNoSep = errors.New("missing separator")

type scanner func(string) (linker, string, error) // TODO consider upgrading to []linker

var commands = map[byte]scanner{
	'x': scanX,
	'y': scanY,
	'g': scanG,
	'v': scanV,
	'p': scanP,
}

// NOTE not actually a "scanner" due to needing to de-confuse the `type scanner` as noted above.
func scanCommand(s string) ([]linker, error) {
	var lnks []linker
	for len(s) > 0 {
		scan, def := commands[s[0]]
		if !def {
			return nil, fmt.Errorf("unrecognized command %q", s[0])
		}
		lnk, cont, err := scan(s[1:])
		if err != nil {
			return nil, err
		}
		lnks = append(lnks, lnk)
		s = cont
	}
	return lnks, nil
}

//// command scanners

func scanX(s string) (lnk linker, _ string, _ error) {
	switch s[0] {
	case '[':
		return scanXBal('[', ']', s[1:])
	case '{':
		return scanXBal('{', '}', s[1:])
	case '(':
		return scanXBal('(', ')', s[1:])
	case '<':
		return scanXBal('<', '>', s[1:])
	default:
		re, s, err := scanPat(s[0], s[1:])
		if err == nil {
			lnk, err = xcommandReLinker(re)
		}
		return lnk, s, err
	}
}

func scanXBal(start, end byte, s string) (lnk linker, _ string, _ error) {
	inc := false
	if len(s) > 0 && s[0] == end {
		inc = true // TODO consider the usability of this
		s = s[1:]
	}
	lnk, err := xcommandBalLinker(start, end, inc)
	return lnk, s, err
}

func scanY(s string) (lnk linker, _ string, err error) {
	sep := s[0]
	s = s[1:]
	var pats [2]*regexp.Regexp
	for i := 0; len(s) > 0 && i < 2; i++ {
		pats[i], s, err = scanPat(sep, s)
		if err != nil {
			break
		}
	}
	if err == nil {
		lnk, err = ycommandLinker(pats[0], pats[1])
	}
	return lnk, s, err
}

func scanG(s string) (lnk linker, _ string, _ error) {
	re, s, err := scanPat(s[0], s[1:])
	if err == nil {
		lnk, err = gcommandLinker(re, false)
	}
	return lnk, s, err
}

func scanV(s string) (lnk linker, _ string, _ error) {
	re, s, err := scanPat(s[0], s[1:])
	if err == nil {
		lnk, err = gcommandLinker(re, true)
	}
	return lnk, s, err
}

func scanP(s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case '%':
		if len(s) < 3 || s[1] != '"' {
			return nil, s, errors.New("missing format scring to p%")
		}
		s = s[1:]
		var format string
		format, s, err = scanString(s[0], s[1:])
		if err == nil {
			lnk, err = pcommandLinker(format, nil)
		}

	case '"':
		var tmp string
		tmp, s, err = scanString(s[0], s[1:])
		if err == nil {
			lnk, err = pcommandLinker("", []byte(tmp))
		}

	default:
		lnk, err = pcommandLinker("", nil)
	}
	return lnk, s, err
}

//// common pieces

func scanDelim(sep byte, r string) (part, rest string, err error) {
	// TODO support escaping
	i := strings.Index(r, string(sep))
	if i < 0 {
		return "", "", errNoSep
	}
	return r[:i], r[i+1:], nil
}

func scanPat(sep byte, r string) (*regexp.Regexp, string, error) {
	pat, rest, err := scanDelim(sep, r)
	if err != nil {
		return nil, "", err
	}

	if sep == '"' || sep == '\'' {
		pat = regexp.QuoteMeta(pat)
	}

	if len(rest) > 0 && rest[0] == 'i' {
		// TODO reconsider the case insensitivity affordance
		pat = "(?i:" + pat + ")"
		rest = rest[1:]
	}

	pat = "(?ms:" + pat + ")"
	re, err := regexp.Compile(pat)
	return re, rest, err
}

func scanString(sep byte, s string) (val, rest string, err error) {
	if val, s, err = scanDelim(sep, s); err == nil {
		val = fmt.Sprintf("%s%s%s", string(sep), val, string(sep))
		val, err = strconv.Unquote(val)
	}
	return val, s, err
}
