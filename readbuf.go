package main

import (
	"bytes"
	"errors"
	"io"
)

const (
	minRead = 64 * 1024 // TODO tune
)

var errNegativeRead = errors.New("readBuf: reader returned negative count from Read")

// ripped from bytes.Buffer
type readBuf struct {
	buf []byte // contents are the bytes buf[off : len(buf)]
	off int    // read at &buf[off], write at &buf[len(buf)]
}

func (rb *readBuf) Advance(n int) {
	rb.off += n
}
func (rb *readBuf) Bytes() []byte { return rb.buf[rb.off:] }
func (rb *readBuf) Len() int      { return len(rb.buf) - rb.off }
func (rb *readBuf) Cap() int      { return cap(rb.buf) }
func (rb *readBuf) Reset() {
	rb.buf = rb.buf[:0]
	rb.off = 0
}

func (rb *readBuf) readMore(r io.Reader) (n int, err error) {
	i := rb.grow(minRead)
	n, err = r.Read(rb.buf[i:cap(rb.buf)])
	if n < 0 {
		panic(errNegativeRead)
	}
	rb.buf = rb.buf[:i+n]
	return n, err
}

func (rb *readBuf) grow(n int) int {
	const maxInt = int(^uint(0) >> 1)

	m := rb.Len()
	// If buffer is empty, reset to recover space.
	if m == 0 && rb.off != 0 {
		rb.Reset()
	}
	// Try to grow by means of a reslice.
	if i, ok := rb.tryGrowByReslice(n); ok {
		return i
	}

	c := cap(rb.buf)
	if n <= c/2-m {
		// We can slide things down instead of allocating a new
		// slice. We only need m+n <= c to slide, but
		// we instead let capacity get twice as large so we
		// don't spend all our time copying.
		copy(rb.buf, rb.buf[rb.off:])
	} else if c > maxInt-c-n {
		panic(bytes.ErrTooLarge)
	} else {
		// Not enough space anywhere, we need to allocate.
		buf := makeSlice(2*c + n)
		copy(buf, rb.buf[rb.off:])
		rb.buf = buf
	}
	// Restore rb.off and len(rb.buf).
	rb.off = 0
	rb.buf = rb.buf[:m+n]
	return m
}

func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(bytes.ErrTooLarge)
		}
	}()
	return make([]byte, n)
}

func (rb *readBuf) tryGrowByReslice(n int) (int, bool) {
	if l := len(rb.buf); n <= cap(rb.buf)-l {
		rb.buf = rb.buf[:l+n]
		return l, true
	}
	return 0, false
}

func (rb *readBuf) Process(cmd command, r io.Reader) error {
	for {
		m, err := rb.readMore(r)
		last := err == io.EOF
		buf := rb.Bytes()
		m, procErr := cmd.Process(buf, last)
		if m > 0 {
			rb.Advance(m)
		}
		if err == nil || last {
			err = procErr
		}
		if err != nil || last {
			return err
		}
	}
}
