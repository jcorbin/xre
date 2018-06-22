# `xre` is to [`sam`][sam] as [`grep`][grep] is to [`ed`][ed]

`xre` exists to bring the awesome power of Rob Pike's Rob Pike's [Structural
Regular Expressions][seregexp] beyond the reach of the `sam` editor. For
maximum Rob Pike (et al) it is written in [Go][go].

**WARNING**: It is still in a primordial / experimental phase, but works well as a proof of concept.

## What?

A short comparison to the grep/ed model:

- a new `x/re/` command extracts structure matched by a regular expression
- ... `x[` `x{` `x(` and `x<` extract a balanced pair of braces
- a new `y/re/` command extracts structure delimited by a regular expression
- ... `y"delim"` extracts structure between occurrences of a static delimiter, e.g. `y"\n"` for classic UNIX line-orientation
- ... `y/start/end/` extracts structure between two regular expressions
- ... `y[` `y{` `y(` and `y<` extract content within a balanced pair of braces
- the `g/re/` command filters the current buffer (as extracted by `x` or `y`) if the given pattern matches
- the `v/re/` command filters the current buffer (as extracted by `x` or `y`) if the given pattern doesn't matches
- the `p` command prints
- ... `p"delim"` prints with a delimiter, e.g. `p"\n"` to return to the warm embrace of classic UNIX tools
- ... `p%"format"` prints with a format pattern, e.g. `p"%q\n"` is particularly useful while developing an xre program

## Why?

Loosely quoting from Rob Pike's [Structural Regular Expressions][seregexp]:

> ...if the interesting quantum of information isn’t a line, most of the (UNIX) tools don’t help, or at best do poorly

### Example: counting Go heap allocations

For example, it is sometimes useful to deal with things like paragraphs (bytes
that are delimited by a blank line, i.e. `"\n\n"`). For maximal self reference,
such a data set can be had from your nearest Go program form either its
[`/debug/pprof/heap?debug=1`][httpPprof] endpoint, or by calling
[`pprof.Lookup("heap").WriteTo(f, 1)`][runtimePprof] yourself.

For example, the following xre program extracts just the allocation bytes from
heap allocations involving a call to `bytes.makeSlice` (i.e. when a
`bytes.Buffer` needs to grow):

```
xre 'y"\n\n"' v/bytes.makeSlice/ 'y"\n"' 'v/^#|^$/' 'x[x/^\d: (\d+)/' 'p"\n"'
```

Breaking down the above command
- extract paragraphs (buffers defined delimited by blank lines)
- keep only the paragraphs that mention "bytes.makeSlice"
- extract lines within those paragraphs
- and keep only the lines that aren't blank and don't start with a `"#"`
- on those lines, extract the contents of the first balanced `[ ]` pair
- and then extract the "MMM" in a "NNN: MMM" match within it
- finally, print those numbers delimited by new lines (the classic UNIX paradigm)

As always, summing a stream of numbers is left as an exercise to the reader.

[sam]: https://en.wikipedia.org/wiki/Sam_(text_editor)
[grep]: https://en.wikipedia.org/wiki/Grep
[ed]: https://en.wikipedia.org/wiki/Ed_(text_editor)
[seregexp]: http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf
[go]: http://golang.org/
[runtimePprof]: https://golang.org/pkg/runtime/pprof
[httpPprof]: https://golang.org/pkg/net/http/pprof/
