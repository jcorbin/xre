package xre

import "fmt"

type predicate interface {
	test([]byte) bool
}

type anyPredicate []predicate
type allPredicate []predicate
type notPredicate struct{ predicate }

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

func (np notPredicate) test(buf []byte) bool { return !np.predicate.test(buf) }

func (ap anyPredicate) test(buf []byte) bool {
	for _, p := range ap {
		if p.test(buf) {
			return true
		}
	}
	return false
}

func (ap allPredicate) test(buf []byte) bool {
	for _, p := range ap {
		if !p.test(buf) {
			return false
		}
	}
	return true
}

// TODO apply predicates to groups
// - buffer each record
// - until we know whether wanted
// - then (start) sending records
