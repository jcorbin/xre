package xre_test

import (
	"errors"
	"testing"
)

func Test_matchProcessor_read_errors(t *testing.T) {
	cmdTestCases{

		{name: "initial error",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				errors.New("bang"),
				"bob lob law, bla blab bib.",
			),
			err: "bang",
		},

		{name: "error between words",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law,",
				errors.New("bang"),
				" bla blab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				`),
			err: "bang",
		},

		{name: "mid-word error 1",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"b",
				errors.New("bang"),
				"ob lob law, bla blab bib.",
			),
			out: []byte{},
			err: "bang",
		},

		{name: "mid-word error 2",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob l",
				errors.New("bang"),
				"ob law, bla blab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				`),
			err: "bang",
		},

		{name: "mid-word error 3",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob l",
				errors.New("bang"),
				"aw, bla blab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				`),
			err: "bang",
		},

		{name: "mid-word error 4",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law",
				errors.New("bang"),
				", bla blab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				`),
			err: "bang",
		},

		{name: "mid-word error 5",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law, b",
				errors.New("bang"),
				"la blab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				`),
			err: "bang",
		},

		{name: "mid-word error 6",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law, bla bl",
				errors.New("bang"),
				"ab bib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				"blab"
				"bib"
				`),
			err: "bang",
		},

		{name: "mid-word error 7",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law, bla blab b",
				errors.New("bang"),
				"ib.",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				"blab"
				"bib"
				`),
			err: "bang",
		},

		{name: "mid-word error 8",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law, bla blab bib",
				errors.New("bang"),
				".",
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				"blab"
				"bib"
				`),
			err: "bang",
		},

		{name: "final error",
			cmd: `x/\w+/ p%"%q\n"`,
			in: readFixture(
				"bob lob law, bla blab bib.",
				errors.New("bang"),
			),
			out: stripBlockSpace(`
				"bob"
				"lob"
				"law"
				"bla"
				"blab"
				"bib"
				`),
			err: "bang",
		},
	}.run(t)
}
