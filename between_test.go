package xre_test

import "testing"

func Test_integration(t *testing.T) {
	cmdTestCases{
		{
			name: "finding and cutting through the blas",
			cmd:  `y"\n\n" g/bla/ y"\n" v/bla/ j, p"\n"`,
			in: stripBlockSpace(`
			9 440
			bla
			bla
			foo
			bar

			10 100
			lab
			lab
			shepherd
			heeler

			12 1302
			bla
			bla
			bla
			bob
			lob
			law
			`),
			out: stripBlockSpace(`
			9 440,foo,bar
			12 1302,bob,lob,law
			`),
		},
	}.run(t)
}

func Test_between(t *testing.T) {
	cmdTestCases{
		{name: "line splitting",
			cmd: `y"\n" p%"%q\n"`,
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

		{name: "paragraph splitting",
			cmd: `y"\n\n" p%"%q\n"`,
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

		{name: "lines within paragraphs",
			cmd: `y"\n\n" y"\n" p%"%q\n"`,
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

		{name: "words in lines in paragraphs",
			cmd: `y"\n\n" y"\n" y/\s+/ p%"%q\n"`,
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

		{name: "between marker lines",
			cmd: `y/\n*--- MARK ---\n+/ p%"%q\n"`,
			in: stripBlockSpace(`
			--- MARK ---
			bla bla
			bla

			--- MARK ---
			what's all
			this
			then?


			--- MARK ---

			the king is dead
			long live the king

			`),
			out: stripBlockSpace(`
			""
			"bla bla\nbla"
			"what's all\nthis\nthen?"
			"the king is dead\nlong live the king\n\n"
			`),
		},
	}.run(t)
}
