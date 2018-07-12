package xre

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
)

// Environment abstracts command runtime context; currently this only means
// where output goes.
type Environment interface {
	Inputs() <-chan Input
	Default() Processor
	Close() error
	// Create(name string) (processor, error) TODO
	// Printf(format string, args ...interface{}) TODO
}

// Input represents either a successfully acquired input stream, or a failure
// to acquire one under an Environment.
type Input struct {
	io.ReadCloser
	Err error
}

// EnvDefaultRead can be implemented by an environment to provide default
// toplevel semantics; this allows processors (e.g. the g and v commands'
// filter) to be used directly.
type EnvDefaultRead interface {
	Environment

	DefaultReader(Processor) io.ReaderFrom
}

type _nullEnv struct{}

// FileEnv is an Environment backed directly by files; there may be a default
// provided input file, and output goes into a single provided file.
type FileEnv struct {
	DefaultInfile  *os.File
	DefaultOutfile *os.File

	bufw *bufio.Writer
	defp Processor
	ins  chan Input
}

// Stdenv is the default expected Environment that defaults to reading from
// os.Stdin and writes to os.Stdout.
var Stdenv = FileEnv{
	DefaultInfile:  os.Stdin,
	DefaultOutfile: os.Stdout,
}

// Inputs returns a channel that will contain any caller specified inputs.
// Returns the same channel every time, therefore it only makes sense to run a
// single command under a FileEnv.
//
// Inputs may be specified in one of two ways:
// - if the caller calls AddInput() one or more times before the first call to
//   Inputs(), then any such added inputs will be used
// - otherwise AddInput(DefaultInfile, nil) is called, and then CloseInputs()
//   so that no other inputs maybe added
func (fe *FileEnv) Inputs() <-chan Input {
	if fe.ins == nil {
		fe.AddInput(fe.DefaultInfile, nil)
		fe.CloseInputs()
	}
	return fe.ins
}

// AddInput allocates an inputs channel, and adds any non-nil file or error as
// given. The channel is allocated with minimal capacity (currently 1), and so
// will block to avoid eagerly opening a huge backlog of inputs.
//
// If the caller intends to add an arbitrary number of inputs (e.g. from some
// user-given list), it should do so in a separate goroutine from the one
// running the command. This also means that it should at least call
// AddInput(nil, nil) before running the command, if not open and add the first
// input first.
func (fe *FileEnv) AddInput(f *os.File, err error) {
	if fe.ins == nil {
		fe.ins = make(chan Input, 1)
	}
	if err != nil {
		fe.ins <- Input{nil, err}
	} else if f != nil {
		fe.ins <- Input{f, nil}
	}
}

// CloseInputs closes any input channel, allocating it if necessary first so
// that any future AddInput or CloseInputs call will panic.
func (fe *FileEnv) CloseInputs() {
	if fe.ins == nil {
		fe.ins = make(chan Input, 0)
	}
	close(fe.ins)
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

func (ne _nullEnv) Inputs() <-chan Input { return nil }
func (ne _nullEnv) Default() Processor   { return writer{ioutil.Discard} }
func (ne _nullEnv) Close() error         { return nil }

// BufEnv is an Environment that reads input from an in-memory buffer, and
// collects all output in another in-memory buffer; useful mainly for testing.
type BufEnv struct {
	Input         bytes.Buffer
	DefaultOutput bytes.Buffer

	ins chan Input
}

// Reset all input and output state, preparing the BufEnv for re-use under a
// new command/input pair.
func (be *BufEnv) Reset() {
	be.Input.Reset()
	be.DefaultOutput.Reset()
	be.ins = nil
}

// RunProcessor runs the given Processor with the given input bytes, and
// returns any output bytes and processing error.
func (be *BufEnv) RunProcessor(proc Processor, input []byte) (out []byte, err error) {
	be.Reset()
	err = proc.Process(input, true)
	return be.DefaultOutput.Bytes(), err
}

// RunReaderFrom runs the given io.ReaderFrom with any given input io.Readers,
// and returns any output bytes and processing error. If no inputs are given,
// then then rf is run only once with an empty io.Reader stream.
func (be *BufEnv) RunReaderFrom(rf io.ReaderFrom, inputs ...io.Reader) (out []byte, err error) {
	be.Reset()
	if len(inputs) > 0 {
		be.SetInputs(inputs...)
	}
	err = RunReaderFrom(rf, be)
	return be.DefaultOutput.Bytes(), err
}

// Inputs returns a channel which will contain a single Input, wrapping the
// BufEnv.Input value. It returns the same channel every time, until Reset is
// called.
func (be *BufEnv) Inputs() <-chan Input {
	if be.ins == nil {
		be.SetInputs(&be.Input)
	}
	return be.ins
}

// SetInputs stores the given io.Readers (upgraded or adapted to
// io.ReadCloser) for future reception under Inputs().
func (be *BufEnv) SetInputs(rs ...io.Reader) {
	if be.ins != nil {
		panic("BufEnv inputs already set")
	}
	be.ins = make(chan Input, len(rs))
	for _, r := range rs {
		if rc, ok := r.(io.ReadCloser); ok {
			be.ins <- Input{rc, nil}
		} else {
			be.ins <- Input{ioutil.NopCloser(r), nil}
		}
	}
	close(be.ins)
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
