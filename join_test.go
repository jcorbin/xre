package xre_test

import "testing"

func Test_join(t *testing.T) {
	cmdTestCases{
		{
			name: "word un-segmentation",
			cmd:  `x/\w+/ j`,
			proc: `x/\w+/ p`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: []byte("ablewasIereIsawelbathequickbrownfoxjumpsoverthelazyhound"),
		},

		{
			name: "joined words",
			cmd:  `x/\w+/ j" "`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: []byte("able was I ere I saw elba the quick brown fox jumps over the lazy hound"),
		},

		{
			name: "elaborately joined words",
			cmd:  `x/\w+/ j", "`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: []byte("able, was, I, ere, I, saw, elba, the, quick, brown, fox, jumps, over, the, lazy, hound"),
		},

		{
			name: "word un-segmentation, with paragraph structure",
			cmd:  `y/\n\n/ x/\w+/ j p"\n"`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: stripBlockSpace(`
			ablewasIereIsawelba
			thequickbrownfoxjumpsoverthelazyhound
			`),
		},

		{
			name: "words in paras (comma sep)",
			cmd:  `y/\n\n/ x/\w+/ j, p"\n"`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: stripBlockSpace(`
			able,was,I,ere,I,saw,elba
			the,quick,brown,fox,jumps,over,the,lazy,hound
			`),
		},

		{
			name: "elaborately joined words, with paragraph structure",
			cmd:  `y/\n\n/ x/\w+/ j", " p"\n"`,
			in: stripBlockSpace(`
			able was I
			ere
			I saw elba.

			the quick brown
			fox jumps over
			the lazy  hound.
			`),
			out: stripBlockSpace(`
			able, was, I, ere, I, saw, elba
			the, quick, brown, fox, jumps, over, the, lazy, hound
			`),
		},
	}.run(t)
}
