package main

import (
	"regexp"
	"testing"
)

func Test_betweenDelim(t *testing.T) {
	withTestSink(t, func(out command, run func(tc cmdTestCase)) {

		for _, tc := range []cmdTestCase{
			{
				name: "line splitting",
				cmd: betweenDelimSplit{
					split: lineSplitter(1),
					next:  &fmter{fmt: "%q\n", next: out},
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
					next:  &fmter{fmt: "%q\n", next: out},
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
						next:  &fmter{fmt: "%q\n", next: out},
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

			{
				name: "words in lines in paragraphs",
				cmd: betweenDelimSplit{
					split: lineSplitter(2),
					next: betweenDelimSplit{
						split: lineSplitter(1),
						next: betweenDelimRe{
							pat:  regexp.MustCompile(`\s+`),
							next: &fmter{fmt: "%q\n", next: out},
						},
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
		} {
			run(tc)
		}
	})
}
