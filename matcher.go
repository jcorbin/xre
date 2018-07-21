package xre

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
	pendLoc  bool
	priorLoc [3]int
	next     Processor
}

func (mp matchProcessor) String() string {
	return fmt.Sprintf("%v %v", mp.matcher, mp.next)
}

func (mp *matchProcessor) Process(buf []byte, last bool) error {
	if buf == nil {
		return mp.next.Process(nil, last)
	}
	mp.pendLoc = false
	mp.priorLoc = [3]int{0, 0, 0}
	return mp.buf.ProcessIn(buf, mp.run)
}

func (mp *matchProcessor) ReadFrom(r io.Reader) (int64, error) {
	mp.pendLoc = false
	mp.priorLoc = [3]int{0, 0, 0}
	return mp.buf.ProcessFrom(r, mp.run)
}

func (mp *matchProcessor) run(buf *readBuf) error {
	// TODO could allow full control upgrade ala
	// type matchRunner interface {
	// 	runMatch(mp *matchProcessor)
	// }

	berr := buf.Err()
	for {
		off := mp.offset()
		if off >= len(mp.buf.buf) {
			break
		}
		buf := mp.buf.buf[off:]
		if err := mp.matcher.match(mp, buf); err != nil {
			// matcher failed
			_ = mp.procPrior(false)
			return err
		} else if newOff := mp.offset(); newOff == off {
			// no progress
			break
		} else if newOff == len(mp.buf.buf) {
			// matcher consumed entire buffer
			if berr != io.EOF &&
				mp.pendLoc &&
				mp.priorLoc[1] == mp.priorLoc[2] {
				// Forget pending loc at end of buffer so that we get a
				// chance to match it with more content next time.
				mp.pendLoc = false
				mp.priorLoc = [3]int{0, 0, 0}
			}
			break
		}
	}
	if berr == io.EOF {
		return mp.flush()
	}
	if berr != nil {
		return mp.procPrior(false)
	}
	return nil
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

func (mp *matchProcessor) procPrior(last bool) error {
	var err error
	advance, prior := mp.prior()
	if last || prior != nil {
		err = mp.yield(prior, last)
	}
	mp.buf.Advance(advance)
	return err
}

func (mp *matchProcessor) pushLoc(start, end, next int) error {
	err := mp.procPrior(false)
	mp.buf.Advance(start)
	if err == nil {
		mp.pendLoc = true
		mp.priorLoc = [3]int{0, end - start, next - start}
	}
	return err
}

func (mp *matchProcessor) flush() error {
	if !mp.pendLoc {
		return nil
	}
	mp.flushed = true
	return mp.procPrior(true)
}

func (mp *matchProcessor) flushTrailer() error {
	if !mp.pendLoc {
		return nil
	}
	if err := mp.procPrior(false); err != nil {
		return err
	}
	if mp.buf.Len() == 0 {
		return nil
	}
	token := mp.buf.Bytes()
	err := mp.yield(token, true)
	mp.buf.Advance(len(token))
	return err
}

func (mp *matchProcessor) yield(token []byte, last bool) error {
	return mp.next.Process(token, last)
}
