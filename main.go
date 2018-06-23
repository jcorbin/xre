// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"bufio"
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

	cmd, err := compileCommands(flag.Args(), w)
	if err != nil {
		return err
	}

	return withProf(func() error {
		return runCommand(cmd, os.Stdin, useMmap)
	})
}
