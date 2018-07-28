package xre

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func scanP(s string) (Command, string, error) {
	if len(s) == 0 {
		return writer{}, s, nil
	}
	switch c := s[0]; c {
	case '%':
		if len(s) < 3 || s[1] != '"' {
			return nil, s, errors.New("missing format string to p%")
		}
		fmt, s, err := scanString(s[1], s[2:])
		if err != nil {
			return nil, s, err
		}
		return ProtoCommand{printFormat(fmt)}, s, nil

	case '"':
		delim, s, err := scanString(s[0], s[1:])
		if err != nil {
			return nil, s, err
		}
		return ProtoCommand{printDelim(delim)}, s, nil

	default:
		return nil, s, fmt.Errorf("unrecognized p command")
	}
}

type printFormat string
type printDelim string

type fmtProc struct {
	fmt  string
	tmp  bytes.Buffer
	next Processor
}

type delimProc struct {
	delim []byte
	tmp   bytes.Buffer
	next  Processor
}

type writer struct {
	w io.Writer
}

type fmtWriter struct {
	fmt string
	writer
}

type delimWriter struct {
	delim []byte
	writer
}

func (p printFormat) Create(next Processor) Processor {
	switch impl := next.(type) {
	case writer:
		return fmtWriter{string(p), impl}
	case delimWriter:
		return fmtWriter{string(p) + string(impl.delim), impl.writer}
	}
	return &fmtProc{fmt: string(p), next: next}
}

func (p printDelim) Create(next Processor) Processor {
	switch impl := next.(type) {
	case writer:
		return delimWriter{[]byte(p), impl}
	case delimWriter:
		return delimWriter{append([]byte(p), impl.delim...), impl.writer}
	}
	return &delimProc{delim: []byte(p), next: next}
}

func (wr writer) Create(nc Command, env Environment) (Processor, error) {
	next, err := createProcessor(nc, env)
	return next, err
}

func (fp *fmtProc) Process(buf []byte, last bool) error {
	if buf == nil {
		return fp.next.Process(nil, last)
	}
	fp.tmp.Reset()
	_, _ = fmt.Fprintf(&fp.tmp, fp.fmt, buf)
	return fp.next.Process(fp.tmp.Bytes(), last)
}

func (dp *delimProc) Process(buf []byte, last bool) error {
	if buf == nil {
		return dp.next.Process(nil, last)
	}
	dp.tmp.Reset()
	_, _ = dp.tmp.Write(buf)
	_, _ = dp.tmp.Write(dp.delim)
	return dp.next.Process(dp.tmp.Bytes(), last)
}

func (wr writer) Process(buf []byte, last bool) error {
	if buf == nil {
		return nil
	}
	_, err := wr.w.Write(buf)
	return err
}

func (fw fmtWriter) Process(buf []byte, last bool) error {
	if buf == nil {
		return nil
	}
	_, err := fmt.Fprintf(fw.w, fw.fmt, buf)
	return err
}

func (dw delimWriter) Process(buf []byte, last bool) error {
	if buf == nil {
		return nil
	}
	_, err := dw.w.Write(buf)
	if err == nil {
		_, err = dw.w.Write(dw.delim)
	}
	return err
}

// ReadFrom copies data directly from the given reader to the wrapped writer.
// Also implements for fmtWriter and delimWriter by embedding, so that they
// degrade to ignoring the format/delim request when streaming (rather than
// quote or format arbitrarily-sized read chunks).
func (wr writer) ReadFrom(r io.Reader) (n int64, err error) {
	return io.Copy(wr.w, r)
}

func (p printFormat) String() string  { return fmt.Sprintf("p%%%q", string(p)) }
func (p printDelim) String() string   { return fmt.Sprintf("p%q", string(p)) }
func (fp fmtProc) String() string     { return fmt.Sprintf("p%%%q %v", fp.fmt, fp.next) }
func (dp delimProc) String() string   { return fmt.Sprintf("p%q %v", dp.delim, dp.next) }
func (wr writer) String() string      { return "p" }
func (fw fmtWriter) String() string   { return fmt.Sprintf("p%%%q", fw.fmt) }
func (dw delimWriter) String() string { return fmt.Sprintf("p%q", dw.delim) }
