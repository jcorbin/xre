// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"errors"
	"flag"
	"log"
	"os"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

var mainEnv = fileEnv{
	deff: os.Stdout, // TODO support redirection
}

func run() error {
	flag.Parse()

	// TODO SIGPIPE handler

	args := flag.Args()
	if len(args) == 0 {
		// TODO default to just print? (i.e. degenerate to cat?)
		return errors.New("no command(s) given")
	}
	cmd, err := parseCommand(args[0])
	if err != nil {
		return err
	}
	args = args[1:]

	if len(args) > 0 {
		return errors.New("reading input from file argument(s) not implemented") // TODO
	}

	return withProf(func() error {
		return runCommand(cmd, os.Stdin, &mainEnv)
	})
}
