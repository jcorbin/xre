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
