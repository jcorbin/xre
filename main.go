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
	cmd, err := parseCommand(args[0], w)
	if err != nil {
		return err
	}
	args = args[1:]

	if len(args) > 0 {
		return errors.New("reading input from file argument(s) not implemented") // TODO
	}

	return withProf(func() error {
		return runCommand(cmd, os.Stdin, useMmap)
	})
}
