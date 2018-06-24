package main

import (
	"fmt"
	"regexp"
)

func scanX(s string) (linker, string, error) {
	if len(s) == 0 {
		return nil, s, fmt.Errorf("empty x command")
	}
	var x linker
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		var open, close byte
		s = s[1:]
		open = c
		close = balancedOpens[c]
		x = xBalLinker(open, close)

	case '/':
		var pat *regexp.Regexp
		var err error
		pat, s, err = scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		x = xReLinker(pat)

	default:
		return nil, s, fmt.Errorf("unrecognized x command")
	}
	return x, s, nil
}

func xBalLinker(start, end byte) linker {
	return func(next command) (command, error) {
		return extractBalanced{start, end, next}, nil
	}
}

func xReLinker(pat *regexp.Regexp) linker {
	return func(next command) (command, error) {
		switch n := pat.NumSubexp(); n {
		case 0:
			return extractRe{pat, next}, nil
		case 1:
			return extractReSub{pat, next}, nil
		default:
			return nil, fmt.Errorf("extraction with %v sub-patterns not supported", n)
		}
	}
}

type extractRe struct {
	pat  *regexp.Regexp
	next command
}

type extractReSub struct {
	pat  *regexp.Regexp
	next command
}

type extractBalanced struct {
	open, close byte
	next        command
}

func (er extractRe) Process(buf []byte, ateof bool) (off int, err error) {
	for err == nil && off < len(buf) {
		loc := er.pat.FindIndex(buf[off:])
		if loc == nil {
			break
		}
		m := buf[off+loc[0] : off+loc[1]] // extracted match
		if off += loc[1]; off < len(buf) {
			_, err = er.next.Process(m, false)
		} else {
			_, err = er.next.Process(m, ateof)
		}
	}
	return off, err
}

func (ers extractReSub) Process(buf []byte, ateof bool) (off int, err error) {
	for err == nil && off < len(buf) {
		locs := ers.pat.FindSubmatchIndex(buf[off:])
		if locs == nil {
			break
		}
		m := buf[off+locs[2] : off+locs[3]] // extracted match
		if off += locs[1]; off < len(buf) {
			_, err = ers.next.Process(m, false)
		} else {
			_, err = ers.next.Process(m, ateof)
		}
	}
	return off, err
}

func (eb extractBalanced) Process(buf []byte, ateof bool) (off int, err error) {
	// TODO escaping? quoting?
	level, start := 0, 0
	for ; err == nil && off < len(buf); off++ {
		switch buf[off] {
		case eb.open:
			if level == 0 {
				start = off
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start : off+1] // extracted match
				_, err = eb.next.Process(m, false)
			}
		}
	}
	return off, err
}

func (er extractRe) String() string {
	return fmt.Sprintf("x/%v/%v", regexpString(er.pat), er.next)
}
func (ers extractReSub) String() string {
	return fmt.Sprintf("x/%v/%v", regexpString(ers.pat), ers.next)
}
func (eb extractBalanced) String() string {
	return fmt.Sprintf("x%s%s%v", string(eb.open), string(eb.close), eb.next)
}
