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
	}.run(t)
}
