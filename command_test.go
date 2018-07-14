package xre

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	BufEnv
}

type cmdTestCase struct {
	name string
	cmd  string
	proc string
	in   []byte
	out  []byte
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
	defer func(prior int) { minRead = prior }(minRead)
	minRead = 1

	cmd, err := ParseCommand(tc.cmd)
	if !assert.NoError(t, err, "couldn't parse command %q", tc.cmd) {
		return
	}

	assert.Equal(t, tc.cmd, fmt.Sprint(cmd), "expected command string to round-trip")

	rf, err := BuildReaderFrom(cmd, te)
	require.NoError(t, err, "unexpected command build error")

	if tc.proc != "" {
		assert.Equal(t, tc.proc, fmt.Sprint(rf), "expected built reader string")
	} else {
		assert.Equal(t, tc.cmd, fmt.Sprint(rf), "expected built reader string to round-trip")
	}

	if proc, ok := rf.(Processor); ok {
		t.Run("xre.Processor mode", func(t *testing.T) {
			te.DefaultOutput.Reset()
			if assert.NoError(t, proc.Process(tc.in, true), "command failed") {
				assert.Equal(t, tc.out, te.DefaultOutput.Bytes(), "expected command output")
			}
		})
	}

	t.Run("io.ReaderFrom mode", func(t *testing.T) {
		te.DefaultOutput.Reset()
		if _, err := rf.ReadFrom(bytes.NewReader(tc.in)); assert.NoError(t, err, "command failed") {
			assert.Equal(t, tc.out, te.DefaultOutput.Bytes(), "expected command output")
		}
	})
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
