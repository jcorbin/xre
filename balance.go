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
	// TODO escaping? quoting?
	level, start := 0, 0
	for off := 0; off < len(buf); off++ {
		switch buf[off] {
		case bb.open:
			if level == 0 {
				start = off + 1
			}
			level++
		case bb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start:off] // extracted match
				if err := bb.next.Process(m, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (eb extractBalanced) Process(buf []byte, last bool) error {
	// TODO escaping? quoting?
	for level, start, off := 0, 0, 0; off < len(buf); off++ {
		switch buf[off] {
		case eb.open:
			if level == 0 {
				start = off
			}
			level++
		case eb.close:
			level--
			if level < 0 {
				level = 0
			} else if level == 0 {
				m := buf[start : off+1] // extracted match
				if err := eb.next.Process(m, false); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
