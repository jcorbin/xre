// Inspired by http://doc.cat-v.org/bell_labs/structural_regexps/se.pdf

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
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
		switch impl := cmd.(type) {
		case io.ReaderFrom:
			_, err := impl.ReadFrom(os.Stdin)
			return err
		}

		if useMmap {
			buf, fin, err := mmap(os.Stdin)
			if err != nil {
				return err
			}
			defer fin()
			_, err = cmd.Process(buf, true)
			return err
		}

		rb := readBuf{buf: make([]byte, 0, minRead)} // TODO configurable buffer size
		return rb.Process(cmd, os.Stdin)
	})
}

func mmap(f *os.File) ([]byte, func() error, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("mmap: cannot stat %q: %v", f.Name(), err)
	}

	size := fi.Size()
	if size <= 0 {
		return nil, nil, fmt.Errorf("mmap: file %q has negative size", f.Name())
	}
	if size != int64(int(size)) {
		return nil, nil, fmt.Errorf("mmap: file %q is too large", f.Name())
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}
	return data, func() error {
		return syscall.Munmap(data)
	}, nil
}
