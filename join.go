package xre

import (
	"bytes"
	"fmt"
	"io"
	"unicode"
)

func scanJ(s string) (Command, string, error) {
	var sep string
	if len(s) > 0 && !unicode.IsSpace(rune(s[0])) {
		switch c := s[0]; c {
		case '"':
			var err error
			sep, s, err = scanString(c, s[1:])
			if err != nil {
				return nil, s, err
			}
		default:
			sep = s[:1]
			s = s[1:]
		}
	}
	switch len(sep) {
	case 0:
		return ProtoCommand{join{}}, s, nil
	case 1:
		return ProtoCommand{joinByte(sep[0])}, s, nil
	default:
		return ProtoCommand{joinString(sep)}, s, nil
	}
}

type join struct{}
type joinByte byte
type joinString string

type joinProc struct {
	tmp  bytes.Buffer
	next Processor
}

type joinByteProc struct {
	sep  joinByte
	tmp  bytes.Buffer
	next Processor
}

type joinByteWriter struct {
	sep   joinByte
	first bool
	w     io.Writer
	bw    byteWriter
}

type joinStringProc struct {
	sep  joinString
	tmp  bytes.Buffer
	next Processor
}

type joinStringWriter struct {
	sep   joinString
	first bool
	tmp   bytes.Buffer
	w     io.Writer
	sw    stringWriter
}

type byteWriter interface {
	WriteByte(c byte) error
}

type stringWriter interface {
	WriteString(s string) (n int, err error)
}

func (j join) Create(next Processor) Processor {
	if wr, ok := next.(writer); ok {
		return wr
	}
	return &joinProc{next: next}
}

func (j joinByte) Create(next Processor) Processor {
	if wr, ok := next.(writer); ok {
		bw, _ := wr.w.(byteWriter)
		return &joinByteWriter{
			sep:   j,
			first: true,
			w:     wr.w,
			bw:    bw,
		}
	}
	return &joinByteProc{sep: j, next: next}
}

func (j joinString) Create(next Processor) Processor {
	if wr, ok := next.(writer); ok {
		sw, _ := wr.w.(stringWriter)
		return &joinStringWriter{
			sep:   j,
			first: true,
			w:     wr.w,
			sw:    sw,
		}
	}
	return &joinStringProc{sep: j, next: next}
}

func (jp *joinProc) Process(buf []byte, last bool) error {
	if buf != nil {
		jp.tmp.Write(buf)
	}
	if !last {
		return nil
	}
	err := jp.next.Process(jp.tmp.Bytes(), true)
	jp.tmp.Reset()
	return err
}

func (jp *joinByteProc) Process(buf []byte, last bool) error {
	if jp.tmp.Len() > 0 {
		jp.tmp.Grow(len(buf) + 1)
		_ = jp.tmp.WriteByte(byte(jp.sep))
	}
	_, _ = jp.tmp.Write(buf)
	if !last {
		return nil
	}
	err := jp.next.Process(jp.tmp.Bytes(), true)
	jp.tmp.Reset()
	return err
}

func (jp *joinStringProc) Process(buf []byte, last bool) error {
	if jp.tmp.Len() > 0 {
		jp.tmp.Grow(len(buf) + len(jp.sep))
		_, _ = jp.tmp.WriteString(string(jp.sep))
	}
	_, _ = jp.tmp.Write(buf)
	if !last {
		return nil
	}
	err := jp.next.Process(jp.tmp.Bytes(), true)
	jp.tmp.Reset()
	return err
}

func (jw *joinByteWriter) writeSep() error {
	if jw.first {
		jw.first = false
		return nil
	}
	if jw.bw != nil {
		return jw.bw.WriteByte(byte(jw.sep))
	}
	_, err := jw.w.Write([]byte{byte(jw.sep)})
	return err
}

func (jw *joinByteWriter) Process(buf []byte, last bool) error {
	err := jw.writeSep()
	if err == nil {
		_, err = jw.w.Write(buf)
	}
	if last {
		jw.first = true
	}
	return err
}

func (jw *joinStringWriter) writeSep() error {
	if jw.first {
		jw.first = false
		return nil
	}
	if jw.sw != nil {
		_, err := jw.sw.WriteString(string(jw.sep))
		return err
	}
	jw.tmp.Reset()
	jw.tmp.WriteString(string(jw.sep))
	_, err := jw.w.Write(jw.tmp.Bytes())
	return err
}

func (jw *joinStringWriter) Process(buf []byte, last bool) error {
	err := jw.writeSep()
	if err == nil {
		_, err = jw.w.Write(buf)
	}
	if last {
		jw.first = true
	}
	return err
}

func (j join) String() string { return "j" }
func (j joinByte) String() string {
	if j == '"' || unicode.IsSpace(rune(j)) {
		return fmt.Sprintf("j%q", string(j))
	}
	return fmt.Sprintf("j%s", string(j))
}
func (j joinString) String() string { return fmt.Sprintf("j%q", string(j)) }

func (jp joinProc) String() string         { return fmt.Sprintf("j %v", jp.next) }
func (jp joinByteProc) String() string     { return fmt.Sprintf("%v %v", jp.sep, jp.next) }
func (jp joinStringProc) String() string   { return fmt.Sprintf("%v %v", jp.sep, jp.next) }
func (jw joinByteWriter) String() string   { return fmt.Sprint(jw.sep) }
func (jw joinStringWriter) String() string { return fmt.Sprint(jw.sep) }
