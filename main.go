// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"log"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	useMmap := false
	flag.BoolVar(&useMmap, "mmap", false, "force using mmap mode rather than streaming")

	flag.Parse()

	// TODO SIGPIPE handler

	var w io.Writer = os.Stdout // TODO support redirection
	w = bufio.NewWriter(w)      // TODO buffering control

	args := flag.Args()
	if len(args) == 0 {
		// TODO default to just print? (i.e. degenerate to cat?)
		return errors.New("no command(s) given")
	}
	lnks, err := scanCommand(args[0])
	if err != nil {
		return err
	}
	args = args[1:]

	if len(args) > 0 {
		return errors.New("reading input from file argument(s) not implemented") // TODO
	}

	var cmd command = writer{w}
	for i := len(lnks) - 1; i >= 0; i-- {
		cmd, err = lnks[i](cmd)
		if err != nil {
			return err
		}
	}

	return withProf(func() error {
		return runCommand(cmd, os.Stdin, useMmap)
	})
}
