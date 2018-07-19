package xre_test

import "testing"

var loremIpsum = stripBlockSpace(`
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nulla vel aliquet
nulla. Morbi bibendum diam vel dolor pharetra tincidunt. Sed ultricies quam
sodales ipsum imperdiet rutrum. Interdum et malesuada fames ac ante ipsum
primis in faucibus. Nulla sed magna hendrerit, tincidunt ante et, tempus quam.
Vestibulum elementum nec tortor et egestas. Nunc purus felis, pretium eu
pharetra eu, egestas vel erat. Integer suscipit lorem sed leo bibendum, eget
cursus dui dictum. Sed risus ligula, posuere nec felis id, semper ultrices
diam. Integer elit tortor, consequat egestas gravida vel, placerat at velit.
Quisque elementum ante ac sapien elementum feugiat. Orci varius natoque
penatibus et magnis dis parturient montes, nascetur ridiculus mus.

Nam molestie turpis in diam venenatis condimentum. Vestibulum quis ipsum nisi.
Vestibulum luctus leo ac enim interdum tincidunt. In tempus lorem sed purus
gravida posuere. Aenean scelerisque interdum maximus. Sed facilisis lorem ut
velit aliquam feugiat. In vel dignissim leo.

Proin felis justo, cursus a metus sit amet, commodo suscipit dolor. Proin a
ligula lacinia, euismod ligula quis, dapibus lacus. Praesent nec ligula
lacinia, sollicitudin nisi ultrices, ornare lectus. Nullam accumsan tellus eget
dapibus accumsan. Duis at efficitur ligula, a imperdiet erat. In condimentum
lacus non massa pretium congue. Vivamus urna est, condimentum eget faucibus sit
amet, convallis ac nisi. Morbi maximus iaculis odio at mattis. Vestibulum
bibendum nisl eget lobortis eleifend. Quisque vitae est in enim facilisis
pharetra id eu nulla. Donec id condimentum lectus, quis tristique arcu. Nunc ut
mattis felis, ut lacinia mi. Quisque malesuada neque vel sem malesuada, sed
volutpat turpis volutpat. Donec elementum vestibulum tellus, a ornare velit
placerat at. Fusce dictum tortor felis, eget interdum dui maximus quis. Etiam
efficitur justo magna, nec aliquet nunc placerat vel.

Quisque euismod egestas dapibus. Nullam tristique congue purus sed fermentum.
Phasellus a urna at lectus dictum porttitor. Pellentesque at libero elementum,
fermentum sem a, maximus dui. Donec ut elementum sem, non eleifend sem. Nulla
ac risus ut neque iaculis bibendum tristique quis mi. Fusce vulputate pulvinar
maximus.

Nunc consequat auctor leo quis scelerisque. Praesent tincidunt eget risus in
dapibus. Phasellus condimentum facilisis sem eu vulputate. Maecenas efficitur
feugiat libero eu gravida. Proin vehicula faucibus sollicitudin. Praesent sit
amet felis erat. Integer sagittis accumsan nunc, eu rhoncus ante convallis
vitae.
`)

func Test_print(t *testing.T) {
	cmdTestCases{
		{name: "degrades to cat",
			cmd: `p`,
			in:  loremIpsum,
			out: loremIpsum,
		},

		{name: "delim + delim",
			cmd:  `y/\n\n/ x/\w+/ p"," p"\n"`,
			proc: `y/\n\n/ x/\w+/ p",\n"`,
			in: stripBlockSpace(`
			This is a sentence, with a comma; it's in a
			paragraph too.
			`),
			out: stripBlockSpace(`
			This,
			is,
			a,
			sentence,
			with,
			a,
			comma,
			it,
			s,
			in,
			a,
			paragraph,
			too,
			`),
		},

		{name: "fmt + delim",
			cmd:  `y/\n\n/ x/\w+/ p%"%q" p"\n"`,
			proc: `y/\n\n/ x/\w+/ p%"%q\n"`,
			in: stripBlockSpace(`
			This is a sentence, with a comma; it's in a
			paragraph too.
			`),
			out: stripBlockSpace(`
			"This"
			"is"
			"a"
			"sentence"
			"with"
			"a"
			"comma"
			"it"
			"s"
			"in"
			"a"
			"paragraph"
			"too"
			`),
		},

		{name: "delim + fmt",
			cmd: `y/\n\n/ x/\w+/ p"," p%"%q\n"`,
			in: stripBlockSpace(`
			This is a sentence, with a comma; it's in a
			paragraph too.
			`),
			out: stripBlockSpace(`
			"This,"
			"is,"
			"a,"
			"sentence,"
			"with,"
			"a,"
			"comma,"
			"it,"
			"s,"
			"in,"
			"a,"
			"paragraph,"
			"too,"
			`),
		},

		{name: "fmt + fmt",
			cmd: `y/\n\n/ x/\w+/ p%"%q" p%"- %s\n"`,
			in: stripBlockSpace(`
			This is a sentence, with a comma; it's in a
			paragraph too.
			`),
			out: stripBlockSpace(`
			- "This"
			- "is"
			- "a"
			- "sentence"
			- "with"
			- "a"
			- "comma"
			- "it"
			- "s"
			- "in"
			- "a"
			- "paragraph"
			- "too"
			`),
		},

		{name: "delim + delim + ...",
			cmd:  `y/\n\n/ x/\w+/ p"," p"\n" y/\n/ p"\n"`,
			proc: `y/\n\n/ x/\w+/ p",\n" y/\n/ p"\n"`,
			in:   []byte(`foo bar`),
			out:  []byte("foo,\nbar,\n"),
		},

		{name: "fmt + delim + ...",
			cmd:  `y/\n\n/ x/\w+/ p%"%q" p"\n" y/\n/ p"\n"`,
			proc: `y/\n\n/ x/\w+/ p%"%q\n" y/\n/ p"\n"`,
			in:   []byte(`foo bar`),
			out:  []byte("\"foo\"\n\"bar\"\n"),
		},

		{name: "delim + fmt + ...",
			cmd:  `y/\n\n/ x/\w+/ p"," p%"%q\n" y/\n/ p"\n"`,
			proc: `y/\n\n/ x/\w+/ p"," p%"%q\n" y/\n/ p"\n"`,
			in:   []byte(`foo bar`),
			out:  []byte("\"foo,\"\n\"bar,\"\n"),
		},
	}.run(t)
}
