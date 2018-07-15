package xre_test

import "testing"

var catAdjacentThings = stripBlockSpace(`
bird
cat
dog
bobcat
fox
cantaloupe
grumpy cat
book
catalog
cab
truck
car
`)

func Test_filter(t *testing.T) {
	// TODO leverage line-oriented defaults once available

	cmdTestCases{
		{
			name: "finding cats",
			cmd:  `y/\n/ g/cat/ p"\n"`,
			in:   catAdjacentThings,
			out: stripBlockSpace(`
			cat
			bobcat
			grumpy cat
			catalog
			`),
		},

		{
			name: "excising cats",
			cmd:  `y/\n/ v/cat/ p"\n"`,
			in:   catAdjacentThings,
			out: stripBlockSpace(`
			bird
			dog
			fox
			cantaloupe
			book
			cab
			truck
			car
			`),
		},
	}.run(t)
}
