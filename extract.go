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
	if x.open == 0 && x.pat == nil {
		return nil, errors.New("empty x command")
	}

	next, err := create(nc, env)
	if err != nil {
		return nil, err
	}

	if x.open != 0 {
		return extractBalanced{x.open, x.close, next}, nil
	}

	switch n := x.pat.NumSubexp(); n {
	case 0:
		return extractRe{x.pat, next}, nil
	case 1:
		return extractReSub{x.pat, next}, nil
	default:
		return nil, fmt.Errorf("extraction with %v sub-patterns not supported", n)
	}
}

// func (x between) String() string TODO needs Create(nil, nil) to work?

type extractRe struct {
	pat  *regexp.Regexp
	next processor
}

type extractReSub struct {
	pat  *regexp.Regexp
	next processor
}

type extractBalanced struct {
	open, close byte
	next        processor
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
