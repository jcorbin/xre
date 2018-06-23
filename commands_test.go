package main

import (
	"bytes"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
)

type cmdTestCase struct {
	name string
	cmd  command
	in   []byte
	out  []byte
}

func Test_betweenDelimSplit(t *testing.T) {
	withTestSink(t, func(out command, run func(tc cmdTestCase)) {
		for _, tc := range []cmdTestCase{
			{
				name: "line splitting",
				cmd: betweenDelimSplit{
					split: lineSplitter(1),
					next:  out,
				},
				// in: []byte("aee\nbee\tdee\ncee"),
				in: stripBlockSpace(`
				aee
				bee	dee
				cee
				`),
				out: stripBlockSpace(`
				"aee"
				"bee\tdee"
				"cee"
				`),
			},

			{
				name: "paragraph splitting",
				cmd: betweenDelimSplit{
					split: lineSplitter(2),
					next:  out,
				},
				in: stripBlockSpace(`
				because:
				- thing
				- thing
				- and another thing

				therefore:
				- red herring
				- wild leap
				`),
				out: stripBlockSpace(`
				"because:\n- thing\n- thing\n- and another thing"
				"therefore:\n- red herring\n- wild leap"
				`),
			},

			{
				name: "lines within paragraphs",
				cmd: betweenDelimSplit{
					split: lineSplitter(2),
					next: betweenDelimSplit{
						split: lineSplitter(1),
						next:  out,
					},
				},
				in: stripBlockSpace(`
				because:
				- thing
				- thing
				- and another thing

				therefore:
				- red herring
				- wild leap
				`),
				out: stripBlockSpace(`
				"because:"
				"- thing"
				"- thing"
				"- and another thing"
				"therefore:"
				"- red herring"
				"- wild leap"
				`),
			},
		} {
			run(tc)
		}
	})
}

// TODO expand this out to be a more integrative harness that parses a command
// string, rather than manually constructed command networks.
func withTestSink(t *testing.T, f func(out command, run func(tc cmdTestCase))) {
	in := bytes.NewReader(nil)
	out := fmtWriter{fmt: "%q\n"}
	var outBuf bytes.Buffer
	f(&out, func(tc cmdTestCase) {
		t.Run(tc.name, func(t *testing.T) {
			in.Reset(tc.in)
			outBuf.Reset()
			out.w = &outBuf
			defer func() {
				out.w = nil
			}()
			if assert.NoError(t, runCommand(tc.cmd, in, false), "unexpected command error") {
				assert.Equal(t, tc.out, outBuf.Bytes(), "expected command output")
			}
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
