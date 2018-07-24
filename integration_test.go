package xre_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"testing"

	"github.com/jcorbin/xre"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration(t *testing.T) {
	testCases := intTestCases{
		{name: "no command"},

		{name: "parse error",
			xreCmd: "bogus",
			check: func(t *testing.T, outb, errb []byte) {
				i := bytes.Index(errb, []byte(`unrecognized command 'b'`))
				if !assert.True(t, i >= 0, "expected error output") {
					log.Printf("err output: %q", errb)
				}
			},
		},

		{name: "extracting from git log",
			sysCmd: []string{"git", "log", "--decorate", "--abbrev-commit", "HEAD~40.."},
			xreCmd: `y/^commit\s+/ x/^[a-f0-9]+/ p%"%q\n"`,
			check: func(t *testing.T, outb []byte) {
				lines := bytes.Split(outb, []byte("\n"))
				if i := len(lines) - 1; len(lines[i]) == 0 {
					lines = lines[:i]
				}
				assert.Equal(t, 40, len(lines), "expected to extract 40 commit lines")
			},
		},

		{name: "searching from git log (no match)",
			sysCmd: []string{"git", "log", "--decorate", "--abbrev-commit", "HEAD~40.."},
			xreCmd: `y/^commit\s+/ g/John Jacob Jingleheimer-Schmidt/ p%"%q\n"`,
			check: func(t *testing.T, outb []byte) {
				assert.Equal(t, []byte{}, outb)
			},
		},

		{name: "all imports",
			sysCmd: []string{"find", ".", "-name", "*.go"},
			xreCmd: `x/import \((.+?)\)/s y/\n/ x/"(.+?)"/ p"\n"`,
			listIn: true,
			check: func(t *testing.T, outb []byte) {
				counts := countSep(outb, []byte("\n"))
				for _, k := range []string{
					"bufio",
					"bytes",
					"errors",
					"flag",
					"fmt",
					"github.com/jcorbin/xre",
					"github.com/jcorbin/xre/internal/cmdutil",
					"github.com/stretchr/testify/assert",
					"github.com/stretchr/testify/require",
					"io",
					"io/ioutil",
					"log",
					"os",
					"os/exec",
					"path/filepath",
					"regexp",
					"runtime/pprof",
					"runtime/trace",
					"sort",
					"strconv",
					"strings",
					"sync",
					"testing",
					"unicode",
				} {
					_, def := counts[k]
					assert.True(t, def, "expected output key %q", k)
					delete(counts, k)
				}
				var extra []string
				for k := range counts {
					extra = append(extra, k)
				}
				sort.Strings(extra)
				assert.Equal(t, []string(nil), extra, "unexpected extra output keys")
			},
		},
	}

	if t.Run("inproc", func(t *testing.T) {
		testCases.run(t, intTestCase.runInproc)
	}) {
		require.NoError(t, buildCmd(t), "unable to build integration test binary")
		t.Run("built cmd", func(t *testing.T) {
			testCases.run(t, intTestCase.runExcmd)
		})
	}
}

func countSep(s, sep []byte) map[string]int {
	counts := make(map[string]int, 64)
	for lines, i := bytes.Split(s, sep), 0; i < len(lines); i++ {
		if line := lines[i]; len(line) > 0 || i+1 < len(lines) {
			counts[string(line)]++
		}
	}
	return counts
}

type intTestCase struct {
	name   string
	sysCmd []string
	xreCmd string
	listIn bool
	check  interface{}
}

type intTestCases []intTestCase

func (tcs intTestCases) run(t *testing.T, f func(tc intTestCase, t *testing.T)) {
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			f(tc, t)
		})
	}
}

func (tc intTestCase) runInproc(t *testing.T) {
	r, rfin := tc.input(t)
	defer rfin()

	opr, opw, err := os.Pipe()
	require.NoError(t, err, "unable to pipe")
	defer func() {
		_ = opr.Close()
		_ = opw.Close()
	}()

	epr, epw, err := os.Pipe()
	require.NoError(t, err, "unable to pipe")
	defer func() {
		_ = epr.Close()
		_ = epw.Close()
	}()

	fe := xre.FileEnv{
		DefaultOutfile: opw,
	}

	if tc.listIn {
		sc := bufio.NewScanner(r)
		require.True(t, sc.Scan(), "must be able to scan at least one input")
		fe.AddInput(os.Open(sc.Text()))
		scerrch := make(chan error, 1)
		go func() {
			for sc.Scan() {
				fe.AddInput(os.Open(sc.Text()))
			}
			fe.CloseInputs()
			scerrch <- sc.Err()
		}()
		defer func() {
			require.NoError(t, <-scerrch, "unexpected listIn scanning error")
		}()
	} else {
		fe.DefaultInfile = r
	}

	errch := make(chan error, 1)
	go func() {
		defer func() {
			_ = opw.Close()
			_ = epw.Close()
		}()
		err := xre.RunCommand(tc.xreCmd, &fe)
		if err != nil {
			_, _ = fmt.Fprintf(epw, "%v\n", err)
		}
		errch <- err
	}()

	tc.runCheck(t, opr, epr, errch)
}

func (tc intTestCase) runExcmd(t *testing.T) {
	r, rfin := tc.input(t)
	defer rfin()

	opr, opw, err := os.Pipe()
	require.NoError(t, err, "unable to pipe")
	defer func() {
		_ = opr.Close()
		_ = opw.Close()
	}()

	epr, epw, err := os.Pipe()
	require.NoError(t, err, "unable to pipe")
	defer func() {
		_ = epr.Close()
		_ = epw.Close()
	}()

	xargs := []string{_builtCmd}
	if tc.listIn {
		xargs = append(xargs, "-l")
	}
	if tc.xreCmd != "" {
		xargs = append(xargs, tc.xreCmd)
	}
	xcmd := exec.Command(xargs[0], xargs[1:]...)
	xcmd.Stdin = r
	xcmd.Stdout = opw
	xcmd.Stderr = epw

	err = xcmd.Start()
	if cerr := opw.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("failed to half-close stdout pipe: %v", cerr)
	}
	if cerr := epw.Close(); err == nil && cerr != nil {
		err = fmt.Errorf("failed to half-close stderr pipe: %v", cerr)
	}
	require.NoError(t, err, "unable to start xre command")

	errch := make(chan error, 1)
	go func() {
		errch <- xcmd.Wait()
	}()
	tc.runCheck(t, opr, epr, errch)
}

func (tc intTestCase) runCheck(t *testing.T, ro, re io.Reader, errch <-chan error) {
	var wg sync.WaitGroup

	f, wantOut := tc.check.(func(t *testing.T, outb []byte))
	g, wantErr := tc.check.(func(t *testing.T, outb, errb []byte))

	var errb []byte
	var errErr error
	wg.Add(1)
	go func() {
		if wantErr {
			errb, errErr = ioutil.ReadAll(re)
		} else {
			sc := bufio.NewScanner(re)
			for sc.Scan() {
				log.Printf("xre command stderr: %s", sc.Bytes())
			}
			errErr = sc.Err()
		}
		wg.Done()
	}()

	var outb []byte
	var outErr error
	wg.Add(1)
	go func() {
		outb, outErr = ioutil.ReadAll(ro)
		wg.Done()
	}()

	err := <-errch
	if _, wantErr := tc.check.(func(t *testing.T, outb, errb []byte)); wantErr {
		assert.Error(t, err, "wanted an error, none found")
	} else {
		assert.NoError(t, err, "unexpected run error")
	}

	wg.Wait()

	require.NoError(t, outErr, "error handling xre command stdout")
	require.NoError(t, errErr, "error handling xre command stderr")

	if tc.check != nil {
		if wantErr {
			g(t, outb, errb)
		} else if wantOut {
			f(t, outb)
		}
	}
}

func (tc intTestCase) input(t *testing.T) (*os.File, func()) {
	if len(tc.sysCmd) == 0 {
		nf, err := os.Open(os.DevNull)
		require.NoError(t, err, "unable to read dev null")
		return nf, func() {
			_ = nf.Close()
		}
	}

	scmd := exec.Command(tc.sysCmd[0], tc.sysCmd[1:]...)

	rcpipe, err := scmd.StdoutPipe()
	require.NoError(t, err, "unable to pipe")
	pipe := rcpipe.(*os.File)

	epipe, err := scmd.StderrPipe()
	require.NoError(t, err, "unable to pipe")

	require.NoError(t, scmd.Start(), "unable to start system command")

	go func() {
		name := tc.sysCmd[0]
		sc := bufio.NewScanner(epipe)
		for sc.Scan() {
			log.Printf("%s stderr: %s", name, sc.Bytes())
		}
	}()

	return pipe, func() {
		require.NoError(t, scmd.Wait(), "system command failed")
	}
}

var (
	_builtCmd    string
	_cmdBuildErr error
)

func buildCmd(t *testing.T) error {
	if _builtCmd == "" && _cmdBuildErr == nil {
		_builtCmd, _cmdBuildErr = filepath.Abs("int_test.bin")
		if _cmdBuildErr == nil {
			cmd := exec.Command("go", "build", "-o", _builtCmd, "./cmd")
			_cmdBuildErr = cmd.Start()
			if _cmdBuildErr == nil {
				_cmdBuildErr = cmd.Wait()
			}
		}
	}
	return _cmdBuildErr
}
