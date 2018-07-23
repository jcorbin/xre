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
	Close() error
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
	// TODO takeover source io.Reader(s)?
}

type _nullEnv struct{}

// FileEnv is an Environment backed directly by files; output goes into a
// single provided file.
type FileEnv struct {
	DefaultOutfile *os.File

	bufw *bufio.Writer
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
		fe.bufw = bufio.NewWriter(fe.DefaultOutfile) // TODO buffering control
		fe.defp = writer{fe.bufw}
	}
	return fe.defp
}

// Close flushes any open output buffer(s) and closes any open files.
func (fe *FileEnv) Close() error {
	if fe.bufw == nil {
		return nil
	}
	err := fe.bufw.Flush()
	if cerr := fe.DefaultOutfile.Close(); err == nil {
		err = cerr
	}
	return err
}

// NullEnv is an Environment that discards all output, useful mainly for
// examining processor structure separate from any real environment.
var NullEnv Environment = _nullEnv{}

func (ne _nullEnv) Default() Processor { return writer{ioutil.Discard} }
func (ne _nullEnv) Close() error       { return nil }

// BufEnv is an Environment that collects all output in an in-memory buffer;
// useful mainly for testing.
type BufEnv struct {
	DefaultOutput bytes.Buffer
}

// Default returns a processor that will write to the DefaultOutput buffer.
func (be *BufEnv) Default() Processor { return writer{&be.DefaultOutput} }

// Close does nothing.
func (be *BufEnv) Close() error { return nil }

// type assertions for fast failure
var (
	// _nullEnv doesn't need one, since it's only use is as an abstract singleton
	_ Environment = &FileEnv{}
	_ Environment = &BufEnv{}
)
