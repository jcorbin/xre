// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"os"

	"github.com/jcorbin/xre"
	"github.com/jcorbin/xre/internal/cmdutil"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() (rerr error) {
	mainEnv := xre.Stdenv // TODO support redirection

	flag.Parse()

	// TODO SIGPIPE handler

	args := flag.Args()

	var prog string
	if len(args) > 0 {
		prog = args[0]
		args = args[1:]
	}

	rcs := make(chan io.ReadCloser, 1)
	if len(args) > 0 {
		return errors.New("reading input from file argument(s) not implemented") // TODO
	}
	rcs <- os.Stdin
	close(rcs)

	return cmdutil.WithProf(func() error {
		return xre.RunCommand(prog, rcs, &mainEnv)
	})
}
