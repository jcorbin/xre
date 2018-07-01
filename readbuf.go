package main

import (
	"bytes"
	"errors"
	"io"
)

const (
	minRead = 64 * 1024 // TODO configurable buffer size
)

var errNegativeRead = errors.New("readBuf: reader returned negative count from Read")

// ripped from bytes.Buffer
type readBuf struct {
	buf []byte // contents are the bytes buf[off : len(buf)]
	off int    // read at &buf[off], write at &buf[len(buf)]
}

func (rb *readBuf) Advance(n int) { rb.off += n }
func (rb *readBuf) Next(n int) []byte {
	off := rb.off + n
	buf := rb.buf[rb.off:off]
	rb.off = off
	return buf
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

// ProcessFrom is a convenience for implementing io.ReaderFrom for a processor;
// see readState.process for details.
func (rb *readBuf) ProcessFrom(
	r io.Reader,
	handle func(rs *readState, final bool) error,
) (n int64, _ error) {
	rb.Reset()
	rs := readState{
		readBuf: rb,
		r:       r,
	}
	return rs.n, rs.process(handle)
}

type readState struct {
	*readBuf
	r   io.Reader
	n   int64
	err error
}

// process reads from the wrapped io.Reader until an error occurs (either a
// read error, or a processing error returned by the handle function). The
// given handle function is called once after every successful read with final
// set to false.
//
// If a read error occurs (maybe but not necessarily io.EOF), and the read
// buffer is not empty, then the handle function is called one last time with
// final set to true.
//
// Any read error or processing error (returned by handle) is returned in the end.
//
// The handler function should (try to) process rs.Bytes() and then call
// rs.Advance() for however many bytes were consumed by the processing. Any
// unconsumed bytes will still be in the buffer next time (unless final is
// true, then there is no next time!)
func (rs *readState) process(handle func(rs *readState, final bool) error) error {
	for rs.err == nil {
		var m int
		m, rs.err = rs.readBuf.readMore(rs.r)
		rs.n += int64(m)
		if rs.err != nil {
			break
		}
		if err := handle(rs, false); err != nil {
			if rs.err != nil {
				err = rs.err
			}
			return err
		}
	}
	var err error
	if rs.err != io.EOF {
		err = rs.err
	}
	if er := handle(rs, true); er != nil && err == nil {
		err = er
	}
	return err
}
