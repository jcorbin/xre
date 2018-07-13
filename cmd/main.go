// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"errors"
	"flag"
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

func run() error {
	mainEnv := xre.Stdenv // TODO support redirection

	flag.Parse()

	// TODO SIGPIPE handler

	args := flag.Args()
	if len(args) == 0 {
		// TODO default to just print? (i.e. degenerate to cat?)
		return errors.New("no command(s) given")
	}

	cmd, err := xre.ParseCommand(args[0])
	if err != nil {
		return err
	}
	args = args[1:]

	if len(args) > 0 {
		return errors.New("reading input from file argument(s) not implemented") // TODO
	}

	rf, err := xre.BuildReaderFrom(cmd, &mainEnv)
	if err != nil {
		return err
	}

	return cmdutil.WithProf(func() error {
		_, err = rf.ReadFrom(os.Stdin)
		return err
	})
}
