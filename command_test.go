package xre_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode"

	"github.com/jcorbin/xre"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	xre.BufEnv
}

type cmdTestCase struct {
	name string
	cmd  string
	proc string
	in   interface{}
	out  []byte
	err  string

	verbose bool
}

type _readFixture []_readFix

type _fixedReader struct {
	fs []_readFix
}

type _readFix struct {
	p   []byte
	err error
}

func readFixture(args ...interface{}) _readFixture {
	fs := make(_readFixture, 0, len(args))
	for _, arg := range args {
		var f _readFix
		switch val := arg.(type) {
		case string:
			f.p = []byte(val)
		case []byte:
			f.p = val
		case error:
			f.err = val
		default:
			panic(fmt.Sprintf("unsupported readFixture arg type %T", arg))
		}
		if i := len(fs) - 1; i >= 0 &&
			fs[i].p == nil &&
			f.err == nil &&
			f.p != nil {
			fs[i].p = f.p
		} else {
			fs = append(fs, f)
		}
	}
	return fs
}

func (f _readFix) String() string {
	var buf bytes.Buffer
	if f.p != nil {
		_, _ = fmt.Fprintf(&buf, "p=%q", f.p)
	}
	if f.err != nil {
		if buf.Len() > 0 {
			_ = buf.WriteByte(',')
		}
		_, _ = fmt.Fprintf(&buf, "err=%v", f.err)
	}
	return buf.String()
}

func (rf _readFixture) Reader() io.Reader {
	return &_fixedReader{append([]_readFix(nil), rf...)}
}

func (fr *_fixedReader) Read(b []byte) (n int, err error) {
	if len(fr.fs) == 0 {
		return 0, io.EOF
	}
	n, err = copy(b, fr.fs[0].p), fr.fs[0].err
	if err != nil {
		fr.fs = nil
		return
	}
	if n < len(fr.fs[0].p) {
		fr.fs[0].p = fr.fs[0].p[n:]
	} else {
		fr.fs = fr.fs[1:]
		if err == nil && len(fr.fs) == 0 {
			err = io.EOF
		}
	}
	return
}

type cmdTestCases []cmdTestCase

func (tcs cmdTestCases) run(t *testing.T) {
	var te testEnv
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.runIn(&te, t)
		})
	}
}

func (tc cmdTestCase) run(t *testing.T) {
	var te testEnv
	tc.runIn(&te, t)
}

func (tc cmdTestCase) runIn(te *testEnv, t *testing.T) {
	// set readBuf size as small as possible to provoke any bugs provoked by
	// buffer advancing earlier.
	defer func(prior int) { xre.MinRead = prior }(xre.MinRead)
	xre.MinRead = 1

	cmd, err := xre.ParseCommand(tc.cmd)
	if !assert.NoError(t, err, "couldn't parse command %q", tc.cmd) {
		return
	}

	assert.Equal(t, tc.cmd, fmt.Sprint(cmd), "expected command string to round-trip")

	rf, err := xre.BuildReaderFrom(cmd, te)
	require.NoError(t, err, "unexpected command build error")

	if tc.proc != "" {
		assert.Equal(t, tc.proc, fmt.Sprint(rf), "expected built reader string")
	} else {
		assert.Equal(t, tc.cmd, fmt.Sprint(rf), "expected built reader string to round-trip")
	}

	if tc.verbose {
		t.Logf("input: %v", tc.in)
	}

	type readable interface {
		Reader() io.Reader
	}

	ra, haveReadable := tc.in.(readable)
	b, haveBytes := tc.in.([]byte)
	require.True(t, haveBytes || haveReadable, "unsupported test case in type %T", tc.in)

	var r io.Reader
	if haveReadable {
		r = ra.Reader()
	} else {
		r = bytes.NewReader(b)
	}
	if tc.verbose {
		r = loggedReader{r, t.Logf}
	}

	if proc, haveProc := rf.(xre.Processor); haveProc && haveBytes {
		t.Run("xre.Processor mode", func(t *testing.T) {
			out, err := te.RunProcessor(proc, b)
			tc.check(t, out, err)
		})
		t.Run("io.ReaderFrom mode", func(t *testing.T) {
			out, err := te.RunReaderFrom(rf, r)
			tc.check(t, out, err)
		})
	} else {
		out, err := te.RunReaderFrom(rf, r)
		tc.check(t, out, err)
	}
}

func (tc cmdTestCase) check(t *testing.T, b []byte, err error) (ok bool) {
	if tc.err == "" {
		ok = assert.NoError(t, err, "unexpected processing error")
	} else {
		ok = assert.EqualError(t, err, tc.err)
	}
	if ok {
		ok = assert.Equal(t, tc.out, b, "expected command output")
	}
	return ok
}

func stripBlockSpace(s string) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	lines := strings.Split(s, "\n")
	indent := ""
	i := 0
	for ; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if len(trimmed) == 0 {
			continue
		}
		if d := len(line) - len(trimmed); d > 0 {
			indent = line[:d]
			line = trimmed
			buf.WriteString(line)
		} else {
			buf.WriteString(line)
		}
		break
	}
	for i++; i < len(lines); i++ {
		trimmed := strings.TrimPrefix(lines[i], indent)
		if len(trimmed) > 0 || i < len(lines)-1 {
			buf.WriteByte('\n')
			buf.WriteString(trimmed)
		}
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

type loggedReader struct {
	io.Reader
	logf func(string, ...interface{})
}

func (lr loggedReader) Read(p []byte) (n int, err error) {
	n, err = lr.Reader.Read(p)
	lr.logf("read => %q, %v", p[:n], err)
	return n, err
}
