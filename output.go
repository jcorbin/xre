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
		return userGivenOutput{printFormat(fmt)}, s, nil

	case '"':
		delim, s, err := scanString(s[0], s[1:])
		if err != nil {
			return nil, s, err
		}
		return userGivenOutput{printDelim(delim)}, s, nil

	default:
		return nil, s, fmt.Errorf("unrecognized p command")
	}
}

type printFormat string
type printDelim string

func procWriter(proc Processor) (io.Writer, bool) {
	switch impl := proc.(type) {
	case *joinByteWriter:
		return impl.w, true
	case *joinStringWriter:
		return impl.w, true
	case fmtWriter:
		return impl.w, true
	case delimWriter:
		return impl.w, true
	case writer:
		return impl.w, true
	}
	return nil, false
}

type userGivenOutput struct{ ProtoProcessor }

func (ugo userGivenOutput) String() string { return fmt.Sprint(ugo.ProtoProcessor) }
func (ugo userGivenOutput) Create(nc Command, env Environment) (Processor, error) {
	next, err := createProcessor(nc, env)
	if err != nil {
		return nil, err
	}
	if nc == nil {
		// nc == nil means we're at the end of user given command chain;
		// override any env-default formatting or delimiting
		if w, ok := procWriter(next); ok {
			next = writer{w}
		}
	}
	return ugo.ProtoProcessor.Create(next), nil
}

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
	case *delimProc:
		p = printFormat(string(p) + string(impl.delim))
		next = impl.next
	case writer:
		return fmtWriter{string(p), impl}
	case delimWriter:
		return fmtWriter{string(p) + string(impl.delim), impl.writer}
	}
	return &fmtProc{fmt: string(p), next: next}
}

func (p printDelim) Create(next Processor) Processor {
	switch impl := next.(type) {
	case *delimProc:
		p = printDelim(string(p) + string(impl.delim))
		next = impl.next
	case writer:
		return delimWriter{[]byte(p), impl}
	case delimWriter:
		return delimWriter{append([]byte(p), impl.delim...), impl.writer}
	}
	return &delimProc{delim: []byte(p), next: next}
}

func (wr writer) Create(nc Command, env Environment) (Processor, error) {
	next, err := createProcessor(nc, env)
	if err != nil || nc != nil {
		return next, err
	}
	if w, ok := procWriter(next); ok {
		wr.w = w
		return wr, nil
	}
	return nil, fmt.Errorf("unable to extract writer from %T", next)
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
