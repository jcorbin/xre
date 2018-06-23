package main

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

type command interface {
	Process(buf []byte, ateof bool) (off int, err error)
}

type filelike interface {
	Name() string
	Stat() (os.FileInfo, error)
	Fd() uintptr
}

func runCommand(cmd command, r io.Reader, useMmap bool) error {
	if f, canMmap := r.(filelike); useMmap && canMmap {
		buf, fin, err := mmap(f)
		if err == nil {
			defer fin()
			_, err = cmd.Process(buf, true)
		}
		return err
	}

	if rf, canReadFrom := cmd.(io.ReaderFrom); canReadFrom {
		_, err := rf.ReadFrom(r)
		return err
	}

	// TODO if (some) commands implement io.Writer, then could upgrade to r.(WriterTo)

	rb := readBuf{buf: make([]byte, 0, minRead)} // TODO configurable buffer size
	return rb.Process(cmd, r)
}

func mmap(f filelike) ([]byte, func() error, error) {
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
