package xre

import "fmt"

type predicate interface {
	test([]byte) bool
}

type predicateProcessor struct {
	predicate
	next Processor
}

func (pp predicateProcessor) String() string {
	return fmt.Sprintf("%v %v", pp.predicate, pp.next)
}

func (pp predicateProcessor) Process(buf []byte, last bool) error {
	if pp.predicate.test(buf) {
		// FIXME may not observe last=true!
		return pp.next.Process(buf, last)
	}
	if buf == nil || last {
		return pp.next.Process(nil, true)
	}
	return nil
}
