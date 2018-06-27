package main

import (
	"bufio"
	"io/ioutil"
	"os"
)

type environment interface {
	Default() processor
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
	// TODO takeover source io.Reader(s)?
}

type fileEnv struct {
	deff *os.File
	defp processor
}

type _nullEnv struct{}

var nullEnv environment = _nullEnv{}

func (fe *fileEnv) Default() processor {
	if fe.defp == nil {
		fe.defp = writer{bufio.NewWriter(fe.deff)} // TODO buffering control
	}
	return fe.defp
}

func (ne _nullEnv) Default() processor {
	return writer{ioutil.Discard}
}
