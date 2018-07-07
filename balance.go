package main

var balancedOpens = map[byte]byte{
	'[': ']',
	'{': '}',
	'(': ')',
	'<': '>',
}

type betweenBalanced struct {
	open, close byte
	next        processor
}

type extractBalanced struct {
	open, close byte
	next        processor
}

func (bb betweenBalanced) Process(buf []byte, last bool) error {
	for {
		if len(buf) == 0 {
			return nil
		}
		if loc, found := scanBalanced(bb.open, bb.close, buf); found {
			if err := bb.next.Process(buf[loc[0]+1:loc[1]-1], false); err != nil {
				return err
			}
			buf = buf[loc[1]:]
		}
	}
}

func (eb extractBalanced) Process(buf []byte, last bool) error {
	for {
		if len(buf) == 0 {
			return nil
		}
		if loc, found := scanBalanced(eb.open, eb.close, buf); found {
			if err := eb.next.Process(buf[loc[0]:loc[1]], false); err != nil {
				return err
			}
			buf = buf[loc[1]:]
		}
	}
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
