package xre

import (
	"fmt"
	"regexp"
)

func scanG(s string) (Command, string, error) {
	pat, s, err := scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return ProtoCommand{regexpFilter{pat}}, s, nil
}

func scanV(s string) (Command, string, error) {
	pat, s, err := scanPat(s[0], s[1:])
	if err != nil {
		return nil, s, err
	}
	return ProtoCommand{regexpNegFilter{pat}}, s, nil
}

type regexpFilter struct{ pat *regexp.Regexp }
type regexpNegFilter regexpFilter

func (fl regexpFilter) test(buf []byte) bool    { return fl.pat.Match(buf) }
func (fn regexpNegFilter) test(buf []byte) bool { return !fn.pat.Match(buf) }

func (fl regexpFilter) Create(next Processor) Processor {
	return &predicateProcessor{predicate: fl, next: next}
}
func (fn regexpNegFilter) Create(next Processor) Processor {
	return &predicateProcessor{predicate: fn, next: next}
}

func (fl regexpFilter) String() string    { return fmt.Sprintf("g%v", regexpString(fl.pat)) }
func (fn regexpNegFilter) String() string { return fmt.Sprintf("v%v", regexpString(fn.pat)) }

/* TODO
func scanAddr(s string) (command, string, error) {
	var ad addr
	var err error

	if ad.start, s, err = scanInt(s); err != nil {
		return nil, s, err
	}

	if len(s) == 0 {
		return nil, s, errors.New("single-address filter not implemented") // TODO
	}
	switch c := s[0]; c {
	case ':':
		if ad.end, s, err = scanInt(s[1:]); err != nil {
			return nil, s, err
		}

	default:
		// TODO support "n1"
		// TODO support "n1,n2,n3,..."
		// TODO support "n~m"
		return nil, s, fmt.Errorf("unsupported address character %q", c)
	}
	return ad, s, nil
}

type addr struct {
	start, end int
}

func (ad addr) Create(nc command, env environment) (processor, error) {
	if ad.start < 0 || ad.end < 0 {
		return nil, errors.New("negative addresses not supported") // TODO
	}
	next, err := createProcessor(nc, env)
	if err != nil {
		return nil, err
	}

	return &addrRange{start: ad.start, end: ad.end, next: next}, nil
}

type addrRange struct {
	start, end, n int
	next          processor
}

func (ar *addrRange) Process(buf []byte, last bool) error {
	n := ar.n + 1
	if last {
		ar.n = 0
	} else {
		ar.n = n
	}
	if ar.start <= n && n <= ar.end {
		return ar.next.Process(buf, n == ar.end)
	}
	return nil
}

func (ad addr) String() string {
	return fmt.Sprintf("%v:%v", ad.start, ad.end)
}
*/
