package main

import (
	"bytes"
	"fmt"
	"io"
)

type fmter struct {
	fmt  string
	tmp  bytes.Buffer
	next command
}

type delimer struct {
	delim []byte
	tmp   bytes.Buffer
	next  command
}

type writer struct {
	w io.Writer
}

type fmtWriter struct {
	fmt string
	w   io.Writer
}

type delimWriter struct {
	delim []byte
	w     io.Writer
}

func (fr *fmter) Process(buf []byte, ateof bool) (off int, err error) {
	fr.tmp.Reset()
	_, _ = fmt.Fprintf(&fr.tmp, fr.fmt, buf)
	return fr.next.Process(fr.tmp.Bytes(), ateof)
}

func (dr *delimer) Process(buf []byte, ateof bool) (off int, err error) {
	dr.tmp.Reset()
	_, _ = dr.tmp.Write(buf)
	_, _ = dr.tmp.Write(dr.delim)
	return dr.next.Process(dr.tmp.Bytes(), ateof)
}

func (wr writer) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = wr.w.Write(buf)
	return len(buf), err
}

func (fw fmtWriter) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = fmt.Fprintf(fw.w, fw.fmt, buf)
	return len(buf), err
}

func (dw delimWriter) Process(buf []byte, ateof bool) (off int, err error) {
	if buf == nil {
		return 0, nil
	}
	_, err = dw.w.Write(buf)
	if err == nil {
		_, err = dw.w.Write(dw.delim)
	}
	return len(buf), err
}
