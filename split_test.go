package xre_test

import "testing"

func Test_splitters(t *testing.T) {
	cmdTestCases{
		{
			name: "comma fields",
			cmd:  `y"\n" y"," p%"%q\n"`,
			in: stripBlockSpace(`
			foo,bar,4
			baz,quz,5
			`),
			out: stripBlockSpace(`
			"foo"
			"bar"
			"4"
			"baz"
			"quz"
			"5"
			`),
		},

		{
			name: "trimmed comma fields",
			cmd:  `y"\n" y","~" " p%"%q\n"`,
			in: stripBlockSpace(`
			foo ,bar,4
			baz,quz ,5
			`),
			out: stripBlockSpace(`
			"foo"
			"bar"
			"4"
			"baz"
			"quz"
			"5"
			`),
		},

		{
			name: "sections",
			cmd:  `y"MARK" p%"%q\n"`,
			in: stripBlockSpace(`
			aee bee
			cee

			MARK

			blargh
			fargh
			gargh

			MARK

			slag slug
			`),
			out: stripBlockSpace(`
			"aee bee\ncee\n\n"
			"\n\nblargh\nfargh\ngargh\n\n"
			"\n\nslag slug\n"
			`),
		},

		{
			name: "trimmed sections",
			cmd:  `y"MARK"~"\n" p%"%q\n"`,
			in: stripBlockSpace(`
			aee bee
			cee

			MARK

			blargh
			fargh
			gargh

			MARK

			slag slug
			`),
			out: stripBlockSpace(`
			"aee bee\ncee"
			"blargh\nfargh\ngargh"
			"slag slug"
			`),
		},
	}.run(t)
}
