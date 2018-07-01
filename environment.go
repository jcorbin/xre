package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
)

type environment interface {
	Default() processor
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
	// TODO takeover source io.Reader(s)?
}

type _nullEnv struct{}

type fileEnv struct {
	deff *os.File
	defp processor
}

type bufEnv struct {
	buf bytes.Buffer
}

var nullEnv environment = _nullEnv{}

func (ne _nullEnv) Default() processor { return writer{ioutil.Discard} }

func (fe *fileEnv) Default() processor {
	if fe.defp == nil {
		fe.defp = writer{bufio.NewWriter(fe.deff)} // TODO buffering control
	}
	return fe.defp
}

func (be *bufEnv) Default() processor { return writer{&be.buf} }
