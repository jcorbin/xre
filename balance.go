package xre

var balancedOpens = map[byte]byte{
	'[': ']',
	'{': '}',
	'(': ')',
	'<': '>',
}

type betweenBalanced struct{ open, close byte }
type extractBalanced betweenBalanced

func (bb betweenBalanced) match(mp *matchProcessor, buf []byte) error {
	if loc, found := scanBalanced(bb.open, bb.close, buf); found {
		return mp.pushLoc(loc[0]+1, loc[1]-1, loc[1])
	}
	return nil
}

func (eb extractBalanced) match(mp *matchProcessor, buf []byte) error {
	if loc, found := scanBalanced(eb.open, eb.close, buf); found {
		return mp.pushLoc(loc[0], loc[1], loc[1])
	}
	return nil
}

func scanBalanced(open, close byte, buf []byte) ([2]int, bool) {
	// TODO escaping? quoting?
	level, start := 0, 0
	for off := 0; off < len(buf); off++ {
		switch buf[off] {
		case open:
			if level == 0 {
				start = off
			}
			level++
		case close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				return [2]int{start, off + 1}, true
			}
		}
	}
	return [2]int{0, 0}, false
}

func (bb betweenBalanced) Create(next Processor) Processor {
	return &matchProcessor{next: next, matcher: bb}
}
func (eb extractBalanced) Create(next Processor) Processor {
	return &matchProcessor{next: next, matcher: eb}
}
