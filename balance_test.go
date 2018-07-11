package main

import "testing"

var fizzBuzzCode = stripBlockSpace(`
import "fmt"

for i := 0; i < 10; i++ {
	any := false
	if i % 3 == 0 {
		any = true
		fmt.Printf("fizz")
	}
	if i % 5 == 0 {
		any = true
		fmt.Printf("buzz")
	}
	if any {
		fmt.Printf("\n")
	}
}
`)

func Test_extract_balanced(t *testing.T) {
	cmdTestCases{
		{
			name: "fizzy code blocks",
			cmd:  `x{ x/{(.*)}/s x{ p%"%q\n"`,
			in:   fizzBuzzCode,
			out: stripBlockSpace(`
			"{\n\t\tany = true\n\t\tfmt.Printf(\"fizz\")\n\t}"
			"{\n\t\tany = true\n\t\tfmt.Printf(\"buzz\")\n\t}"
			"{\n\t\tfmt.Printf(\"\\n\")\n\t}"
			`),
		},
	}.run(t)
}

func Test_between_balanced(t *testing.T) {
	cmdTestCases{
		{
			name: "fizzy code blocks",
			cmd:  `y{ y{ p%"%q\n"`,
			in:   fizzBuzzCode,
			out: stripBlockSpace(`
			"\n\t\tany = true\n\t\tfmt.Printf(\"fizz\")\n\t"
			"\n\t\tany = true\n\t\tfmt.Printf(\"buzz\")\n\t"
			"\n\t\tfmt.Printf(\"\\n\")\n\t"
			`),
		},
	}.run(t)
}
