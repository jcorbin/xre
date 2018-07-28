package xre

import (
	"fmt"
	"io"
)

func matcherCmd(c matchCreator) Command { return matcherCommand{c} }

type matcher interface {
	// TODO consider revoking access to matchProcessor, or
	// hiding behind a minimal interface
	match(mp *matchProcessor, buf []byte) error
}

type matchCreator interface {
	createMatcher(Environment) (matcher, error)
}

type matcherCommand struct {
	matchCreator
}

type matchProcessor struct {
	matcher
	buf      readBuf // TODO embed this also?
	flushed  bool
	pendLoc  bool
	priorLoc [3]int
	next     Processor
}

func (mc matcherCommand) String() string {
	if sr, ok := mc.matchCreator.(fmt.Stringer); ok {
		return sr.String()
	}
	if ma, err := mc.createMatcher(NullEnv); err == nil {
		if sr, ok := ma.(fmt.Stringer); ok {
			return sr.String()
		}
	}
	return fmt.Sprint(mc.matchCreator)
}

func (mc matcherCommand) Create(nc Command, env Environment) (_ Processor, err error) {
	var mp matchProcessor
	mp.matcher, err = mc.createMatcher(env)
	if err != nil {
		return nil, err
	}
	mp.next, err = createProcessor(nc, env)
	if err != nil {
		return nil, err
	}
	return &mp, nil
}

func (mp matchProcessor) String() string {
	return fmt.Sprintf("%v %v", mp.matcher, mp.next)
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
		mp.flushed = false
		mp.pendLoc = true
		mp.priorLoc = [3]int{0, end - start, next - start}
	}
	return err
}

func (mp *matchProcessor) flush() error {
	if mp.flushed {
		return nil
	}
	mp.flushed = true
	return mp.procPrior(true)
}

func (mp *matchProcessor) flushTrailer() error {
	if mp.flushed {
		return nil
	}
	mp.flushed = true
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
