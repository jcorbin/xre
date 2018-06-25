package main

import (
	"bufio"
	"os"
)

type environment interface {
	Default() processor
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
	// TODO takeover source io.Reader(s)?
	// TODO takeover user settings like useMmap?
}

type fileEnv struct {
	deff *os.File
	defp processor
}

func (fe *fileEnv) Default() processor {
	if fe.defp == nil {
		fe.defp = writer{bufio.NewWriter(fe.deff)} // TODO buffering control
	}
	return fe.defp
}

// TODO rename createCommand?
func create(nc command, env environment) (processor, error) {
	if nc == nil {
		return env.Default(), nil
	}
	return nc.Create(nil, env)
}
