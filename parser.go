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
	'a': scanA,
	'p': scanP,
}

// NOTE not actually a "scanner" due to needing to de-confuse the `type scanner` as noted above.
func scanCommand(s string) (lnks []linker, err error) {
	for len(s) > 0 {
		var lnk linker
		var n int
		if n, s, err = scanInt(s); err == nil {
			lnk, s, err = scanAddr(n, s)
		} else if err == errIntExpected {
			err = nil
			if scan, def := commands[s[0]]; def {
				lnk, s, err = scan(s[1:])
			} else {
				err = fmt.Errorf("unrecognized command %q", s[0])
			}
		}
		if err != nil {
			break
		}
		lnks = append(lnks, lnk)
	}
	return lnks, err
}

//// address scanning

func scanAddr(n int, s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case ':':
		s = s[1:]
		var m int
		if m, s, err = scanInt(s); err == nil {
			lnk, err = addrRangeLinker(n, m)
		}

	default:
		// TODO support "n1"
		// TODO support "n1,n2,n3,..."
		// TODO support "n~m"
		err = fmt.Errorf("unsupported address character %q", c)
	}
	return lnk, s, err
}

var errIntExpected = errors.New("missing number")

func scanInt(s string) (n int, _ string, err error) {
	numDigits := 0
	for len(s) > 0 {
		if c := s[0]; '0' <= c && c <= '9' {
			n = 10*n + int(c-'0')
			s = s[1:]
			numDigits++
		} else {
			break
		}
	}
	if numDigits == 0 {
		err = errIntExpected
	}
	return n, s, err
}

//// command scanners

var balancedOpens = map[byte]byte{
	'[': ']',
	'{': '}',
	'(': ')',
	'<': '>',
}

func scanX(s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case '[', '{', '(', '<':
		s = s[1:]
		lnk, err = xBalLinker(c, balancedOpens[c], true)

	case '/':
		var re *regexp.Regexp
		re, s, err = scanPat(c, s[1:])
		if err == nil {
			lnk, err = xReLinker(re)
		}

	default:
		err = fmt.Errorf("unrecognized x command")
	}
	return lnk, s, err
}

func scanY(s string) (lnk linker, _ string, err error) {
	var c byte
	if len(s) > 0 {
		c = s[0]
	}
	switch c {

	case '[', '{', '(', '<':
		s = s[1:]
		lnk, err = xBalLinker(c, balancedOpens[c], false)

	case '/':
		s = s[1:]
		var pats [2]*regexp.Regexp
		for i := 0; len(s) > 0 && i < 2; i++ {
			pats[i], s, err = scanPat(c, s)
			if err != nil {
				break
			}
		}
		if err == nil {
			lnk, err = yReLinker(pats[0], pats[1])
		}

	case '"':
		var delim, cutset string
		delim, s, err = scanString(c, s[1:])
		if err == nil {
			if len(s) > 3 && s[0] == '~' && s[1] == '"' {
				cutset, s, err = scanString(c, s[1:])
			}
		}
		if err == nil {
			lnk, err = yDelimLinker(delim, cutset)
		}

	default:
		// TODO could default to line-delimiting (aka as if y"\n" was given)
		err = fmt.Errorf("unrecognized y command")
	}
	return lnk, s, err
}

func scanG(s string) (linker, string, error) { return scanGV(false, s) }
func scanV(s string) (linker, string, error) { return scanGV(true, s) }
func scanGV(neg bool, s string) (lnk linker, _ string, _ error) {
	re, s, err := scanPat(s[0], s[1:])
	if err == nil {
		lnk, err = gLinker(re, neg)
	}
	return lnk, s, err
}

func scanA(s string) (lnk linker, _ string, err error) {
	lnk, s, err = scanP(s)
	if err == nil {
		lnk = aLinker(lnk)
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
			return nil, s, errors.New("missing format string to p%")
		}
		var format string
		format, s, err = scanString(s[1], s[2:])
		if err == nil {
			lnk, err = pLinker(format, nil)
		}

	case '"':
		var tmp string
		tmp, s, err = scanString(s[0], s[1:])
		if err == nil {
			lnk, err = pLinker("", []byte(tmp))
		}

	default:
		lnk, err = pLinker("", nil)
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
