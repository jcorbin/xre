package main

import "bytes"

type lineSplitter int
type byteSplitter byte
type bytesSplitter []byte

func (ls lineSplitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// TODO CR support

attempt:
	for off := 0; ; {
		i := bytes.IndexByte(data[off:], '\n')
		if i < 0 {
			if atEOF {
				return len(data), data, nil
			}
			return 0, nil, nil
		}
		i += off
		j := i + 1
		for n := 1; n < int(ls) && j < len(data); n++ {
			if data[j] != '\n' {
				off = j
				continue attempt
			}
		}
		return j, data[0:i], nil
	}
}

func (bs byteSplitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, byte(bs)); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func (bss bytesSplitter) Split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, bss); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
