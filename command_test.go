package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	buf bytes.Buffer
}

func (te *testEnv) runTest(t *testing.T, cmd command, in, expected []byte) {
	te.buf.Reset()
	r := bytes.NewReader(in)
	if !assert.NoError(t, runCommand(cmd, r, false), "unexpected command error") {
		return
	}
	assert.Equal(t, expected, te.buf.Bytes(), "expected command output")
}

type cmdTestCase struct {
	name string
	cmd  command
	in   []byte
	out  []byte
}

// TODO expand this out to be a more integrative harness that parses a command
// string, rather than manually constructed command networks.
func withTestSink(t *testing.T, f func(out command, run func(tc cmdTestCase))) {
	var te testEnv
	out := writer{&te.buf}
	f(&out, func(tc cmdTestCase) {
		t.Run(tc.name, func(t *testing.T) {
			te.runTest(t, tc.cmd, tc.in, tc.out)
		})
	})
}

func stripBlockSpace(s string) []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(s)))
	lines := strings.Split(s, "\n")
	i := 0
	indent := ""
	for ; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
		if d := len(line) - len(trimmed); d > 0 {
			indent = line[:d]
			buf.WriteString(trimmed)
			break
		}
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
