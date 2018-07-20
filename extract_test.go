package xre_test

import "testing"

func Test_extract(t *testing.T) {
	cmdTestCases{
		{
			name: "line extraction",
			cmd:  `x/.*\n/ p%"%q\n"`,
			in: stripBlockSpace(`
			aee
			bee	dee
			cee
			`),
			out: stripBlockSpace(`
			"aee\n"
			"bee\tdee\n"
			"cee\n"
			`),
		},

		{
			name: "line extraction (submatch)",
			cmd:  `x/(.*)\n/ p%"%q\n"`,
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
			name: "field extraction",
			cmd:  `x/(.*)\n/ x/^([^\s]+).*$/ p%"%q\n"`,
			in: stripBlockSpace(`
			aee
			bee	dee
			cee
			`),
			out: stripBlockSpace(`
			"aee"
			"bee"
			"cee"
			`),
		},

		{
			name: "word extraction",
			cmd:  `x/\w+/ p"\n"`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: stripBlockSpace(`
			able
			was
			I
			ere
			I
			saw
			elba
			the
			quick
			brown
			fox
			jumps
			over
			the
			lazy
			hound
			`),
		},
	}.run(t)
}
