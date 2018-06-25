package main

import (
	"testing"
)

func Test_betweenDelim(t *testing.T) {
	cmdTestCases{
		{
			name: "line splitting",
			cmd:  `y"\n" p%"%q\n"`,
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
			cmd:  `y"\n\n" p%"%q\n"`,
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
			cmd:  `y"\n\n" y"\n" p%"%q\n"`,
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

		{
			name: "words in lines in paragraphs",
			cmd:  `y"\n\n" y"\n" y/\s+/ p%"%q\n"`,
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
			"-"
			"thing"
			"-"
			"thing"
			"-"
			"and"
			"another"
			"thing"
			"therefore:"
			"-"
			"red"
			"herring"
			"-"
			"wild"
			"leap"
			`),
		},
	}.run(t)
}
