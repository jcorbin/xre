package main

import (
	"fmt"
	"io"
)

type matcher interface {
	// TODO consider revoking access to matchProcessor, or
	// hiding behind a minimal interface
	match(mp *matchProcessor, buf []byte) error
}

type matchProcessor struct {
	matcher
	buf      readBuf // TODO embed this also?
	flushed  bool
	pendLoc  bool
	priorLoc [3]int
	next     processor
}

func createMatcherCommand(m matcher, next processor, env environment) (_ processor, err error) {
	var mp matchProcessor
	mp.matcher = m
	mp.next = next
	return &mp, nil
}

func (mp matchProcessor) String() string {
	return fmt.Sprintf("%v%v", mp.matcher, mp.next)
}

func (mp *matchProcessor) Process(buf []byte, last bool) error {
	mp.flushed = false
	mp.pendLoc = false
	mp.priorLoc = [3]int{0, 0, 0}
	return mp.buf.ProcessIn(buf, mp.run)
}

func (mp *matchProcessor) ReadFrom(r io.Reader) (int64, error) {
	mp.flushed = false
	mp.pendLoc = false
	mp.priorLoc = [3]int{0, 0, 0}
	return mp.buf.ProcessFrom(r, mp.run)
}

func (mp *matchProcessor) run(buf *readBuf) (err error) {
	// TODO could allow full control upgrade ala
	// type matchRunner interface {
	// 	runMatch(mp *matchProcessor)
	// }

	berr := buf.Err()
	if berr == nil || berr == io.EOF {
		for {
			off := mp.offset()
			if off >= len(mp.buf.buf) {
				break
			}
			buf := mp.buf.buf[off:]
			err = mp.matcher.match(mp, buf)
			if err != nil || mp.offset() == off {
				break
			}
		}
	}
	if berr != nil && err == nil && !mp.flushed {
		mp.flushed = true
		advance, token := mp.prior()
		err = mp.yield(token, true)
		mp.buf.Advance(advance)
	}
	return err
}

func (mp *matchProcessor) offset() int {
	off := mp.buf.off
	if mp.pendLoc {
		off += mp.priorLoc[2]
	}
	return off
}

func (mp *matchProcessor) prior() (advance int, token []byte) {
	if mp.pendLoc {
		advance = mp.priorLoc[2]
		token = mp.buf.Bytes()[mp.priorLoc[0]:mp.priorLoc[1]]
		mp.pendLoc = false
		mp.priorLoc = [3]int{0, 0, 0}
	}
	return advance, token
}

func (mp *matchProcessor) flush() (int, error) {
	if mp.flushed {
		return 0, nil
	}
	mp.flushed = true
	advance, token := mp.prior()
	return advance, mp.yield(token, true)
}

func (mp *matchProcessor) procPrior() (int, error) {
	var err error
	advance, prior := mp.prior()
	if prior != nil {
		err = mp.yield(prior, false)
	}
	return advance, err
}

func (mp *matchProcessor) pushLoc(start, end, next int) error {
	advance, err := mp.procPrior()
	mp.buf.Advance(advance + start)
	if err == nil {
		mp.pendLoc = true
		mp.priorLoc = [3]int{0, end - start, next - start}
	}
	return err
}

func (mp *matchProcessor) flushTrailer() error {
	if mp.flushed {
		return nil
	}
	mp.flushed = true
	advance, err := mp.procPrior()
	mp.buf.Advance(advance)
	if end := mp.buf.Len(); end > 0 && err == nil {
		token := mp.buf.Bytes()[0:end]
		err = mp.yield(token, true)
		mp.buf.Advance(end)
	}
	return err
}

func (mp *matchProcessor) yield(token []byte, last bool) error {
	return mp.next.Process(token, last)
}
