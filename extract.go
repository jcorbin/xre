package xre

import (
	"fmt"
	"regexp"
)

func scanX(s string) (Command, string, error) {
	if len(s) == 0 {
		return nil, s, fmt.Errorf("empty x command")
	}
	switch c := s[0]; c {

	case '[', '{', '(', '<':
		s = s[1:]
		return ProtoCommand{extractBalanced{c, balancedOpens[c]}}, s, nil

	case '/':
		pat, s, err := scanPat(c, s[1:])
		if err != nil {
			return nil, s, err
		}
		switch n := pat.NumSubexp(); n {
		case 0:
			return ProtoCommand{extractRe{pat}}, s, nil
		case 1:
			return ProtoCommand{extractReSub{pat}}, s, nil
		default:
			return ProtoCommand{extractReSubs{pat}}, s, nil

		}

	default:
		return nil, s, fmt.Errorf("unrecognized x command")
	}
}

type extractRe struct{ pat *regexp.Regexp }
type extractReSub extractRe
type extractReSubs extractRe

func (er extractRe) match(mp *matchProcessor, buf []byte) error {
	if loc := er.pat.FindIndex(buf); loc != nil {
		return mp.pushLoc(loc[0], loc[1], loc[1])
	}
	return nil
}

func (ers extractReSub) match(mp *matchProcessor, buf []byte) error {
	if locs := ers.pat.FindSubmatchIndex(buf); locs != nil {
		return mp.pushLoc(locs[2], locs[3], locs[1])
	}
	return nil
}

func (erss extractReSubs) match(mp *matchProcessor, buf []byte) error {
	if locs := erss.pat.FindSubmatchIndex(buf); locs != nil {
		off := 0
		for li := 2; li < len(locs); {
			start := locs[li] - off
			li++
			end := locs[li] - off
			li++
			next := locs[1] - off
			if li < len(locs) {
				next = locs[li] - off
			}
			if err := mp.pushLoc(start, end, next); err != nil {
				return err
			}
			off = next
		}
		if err := mp.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (er extractRe) Create(next Processor) Processor {
	return &matchProcessor{matcher: er, next: next}
}
func (ers extractReSub) Create(next Processor) Processor {
	return &matchProcessor{matcher: ers, next: next}
}
func (erss extractReSubs) Create(next Processor) Processor {
	return &matchProcessor{matcher: erss, next: next}
}

func (eb extractBalanced) String() string { return fmt.Sprintf("x%s", string(eb.open)) }
func (er extractRe) String() string       { return fmt.Sprintf("x%v", regexpString(er.pat)) }
func (ers extractReSub) String() string   { return fmt.Sprintf("x%v", regexpString(ers.pat)) }
func (erss extractReSubs) String() string { return fmt.Sprintf("x%v", regexpString(erss.pat)) }
