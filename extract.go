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

	next, err := createProcessor(nc, env)
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

func (er extractRe) Process(buf []byte, last bool) error {
	for off := 0; off < len(buf); {
		loc := er.pat.FindIndex(buf[off:])
		if loc == nil {
			break
		}
		m := buf[off+loc[0] : off+loc[1]] // extracted match
		off += loc[1]
		if err := er.next.Process(m, off >= len(buf)); err != nil {
			return err
		}
	}
	return nil
}

func (ers extractReSub) Process(buf []byte, last bool) error {
	for off := 0; off < len(buf); {
		locs := ers.pat.FindSubmatchIndex(buf[off:])
		if locs == nil {
			break
		}
		m := buf[off+locs[2] : off+locs[3]] // extracted match
		off += locs[1]
		if err := ers.next.Process(m, off >= len(buf)); err != nil {
			return err
		}
	}
	return nil
}

func (eb extractBalanced) Process(buf []byte, last bool) error {
	// TODO escaping? quoting?
	for level, start, off := 0, 0, 0; off < len(buf); off++ {
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
				if err := eb.next.Process(m, false); err != nil {
					return err
				}
			}
		}
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
func (er extractRe) String() string {
	return fmt.Sprintf("x%v%v", regexpString(er.pat), er.next)
}
func (ers extractReSub) String() string {
	return fmt.Sprintf("x%v%v", regexpString(ers.pat), ers.next)
}
func (eb extractBalanced) String() string {
	return fmt.Sprintf("x%s%s%v", string(eb.open), string(eb.close), eb.next)
}
