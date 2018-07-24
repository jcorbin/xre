package xre_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/jcorbin/xre"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration(t *testing.T) {
	for _, tc := range []struct {
		name   string
		sysCmd []string
		xreCmd string
		check  func(t *testing.T, outb []byte)
	}{
		{
			name:   "extracting from git log",
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			scmd := exec.Command(tc.sysCmd[0], tc.sysCmd[1:]...)
			pipe, err := scmd.StdoutPipe()
			require.NoError(t, err, "unable to get system command pipe")
			require.NoError(t, scmd.Start(), "unable to start system command")

			outf, err := ioutil.TempFile("", "")
			require.NoError(t, err, "can't create a tempfile")
			defer func() {
				_ = outf.Close()
				_ = os.Remove(outf.Name())
			}()

			rs := make(chan io.ReadCloser, 1)
			rs <- pipe
			close(rs)

			require.NoError(t, xre.RunCommand(tc.xreCmd, rs, &xre.FileEnv{
				DefaultOutfile: outf,
			}), "failed to run xre command")

			require.NoError(t, scmd.Wait(), "system command failed")

			inf, err := os.Open(outf.Name())
			require.NoError(t, err, "unable to open in file")
			defer func() {
				_ = inf.Close()
			}()

			outb, err := ioutil.ReadAll(inf)
			require.NoError(t, err, "unable to read from tempfile")
			tc.check(t, outb)
		})
	}
}
