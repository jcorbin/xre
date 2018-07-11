package xre

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

type testEnv struct {
	BufEnv
}

func (te *testEnv) runTest(t *testing.T, cmdStr string, in, expected []byte) {
	// set readBuf size as small as possible to provoke any bugs provoked by
	// buffer advancing earlier.
	defer func(prior int) { minRead = prior }(minRead)
	minRead = 1

	cmd, err := ParseCommand(cmdStr)
	if !assert.NoError(t, err, "couldn't parse command %q", cmdStr) {
		return
	}

	assert.Equal(t, cmdStr, fmt.Sprint(cmd), "expected command string to round-trip")

	te.DefaultOutput.Reset()
	r := bytes.NewReader(in)
	if !assert.NoError(t, RunCommand(cmd, r, te), "command failed") {
		return
	}
	assert.Equal(t, expected, te.DefaultOutput.Bytes(), "expected command output")
}

type cmdTestCase struct {
	name string
	cmd  string
	in   []byte
	out  []byte
}

type cmdTestCases []cmdTestCase

func (tcs cmdTestCases) run(t *testing.T) {
	var te testEnv
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			te.runTest(t, tc.cmd, tc.in, tc.out)
		})
	}
}

func (tc cmdTestCase) run(t *testing.T) {
	var te testEnv
	te.runTest(t, tc.cmd, tc.in, tc.out)
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
