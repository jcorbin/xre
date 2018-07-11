package xre

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
)

// Environment abstracts command runtime context; currently this only means
// where output goes.
type Environment interface {
	Default() Processor
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
	// TODO takeover source io.Reader(s)?
}

type _nullEnv struct{}

// FileEnv is an Environment backed directly by files; output goes into a
// single provided file.
type FileEnv struct {
	DefaultOutfile *os.File

	defp Processor
}

// Stdenv is the default expected Environment that writes to os.Stdout.
var Stdenv = FileEnv{
	DefaultOutfile: os.Stdout,
}

// Default returns the default output processor, which will write into the
// provided DefaultOutfile through a buffered writer.
func (fe *FileEnv) Default() Processor {
	if fe.defp == nil {
		fe.defp = writer{bufio.NewWriter(fe.DefaultOutfile)} // TODO buffering control
	}
	return fe.defp
}

// NullEnv is an Environment that discards all output, useful mainly for
// examining processor structure separate from any real environment.
var NullEnv Environment = _nullEnv{}

func (ne _nullEnv) Default() Processor { return writer{ioutil.Discard} }

// BufEnv is an Environment that collects all output in an in-memory buffer;
// useful mainly for testing.
type BufEnv struct {
	DefaultOutput bytes.Buffer
}

// Default returns a processor that will write to the DefaultOutput buffer.
func (be *BufEnv) Default() Processor { return writer{&be.DefaultOutput} }
