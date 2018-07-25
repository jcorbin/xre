package xre_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jcorbin/xre"
)

func TestFileEnv_AddInput(t *testing.T) {
	var fe xre.FileEnv
	fe.AddInput(os.Open(os.DevNull))
	go func() {
		fe.AddInput(os.Open("no/such/path"))
		fe.CloseInputs()
	}()

	ins := fe.Inputs()
	in := <-ins
	assert.NoError(t, in.Err, "unexpected first input error")
	if assert.NotNil(t, in.ReadCloser, "expected first ReadCloser") {
		var buf [128]byte
		_, err := in.ReadCloser.Read(buf[:])
		assert.EqualError(t, err, io.EOF.Error(), "expected EOF read error")
		assert.NoError(t, in.ReadCloser.Close(), "unexpected close error")
	}

	in = <-ins
	assert.Nil(t, in.ReadCloser, "unexpected second ReadCloser")
	assert.EqualError(t, in.Err, "open no/such/path: no such file or directory", "expected second input error")
}

func TestFileEnv_CloseInputs(t *testing.T) {
	var fe xre.FileEnv
	fe.CloseInputs()

	_, ok := <-fe.Inputs()
	assert.False(t, ok, "expected receive to fail")
}

func TestBufEnv_Input(t *testing.T) {
	var be xre.BufEnv
	defer func() {
		assert.NoError(t, be.Close(), "unexpected bufenv close error")
	}()
	_, err := be.Input.WriteString("hello")
	require.NoError(t, err, "unexpected write error")
	out := catBufEnvInputs(t, &be)
	assert.Equal(t, "hello", out, "expected output")
}

func TestBufEnv_SetInputs(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	require.NoError(t, err, "unexpected tempfile error")
	defer func() {
		_ = f.Close()
		_ = os.Remove(f.Name())
	}()

	_, err = f.WriteString("hello")
	require.NoError(t, err, "unexpected write error")
	_, err = f.Seek(0, os.SEEK_SET)
	require.NoError(t, err, "unexpected seek error")

	var be xre.BufEnv
	defer func() {
		assert.NoError(t, be.Close(), "unexpected bufenv close error")
	}()
	be.SetInputs(f)
	assert.Panics(t, func() {
		be.SetInputs(f)
	}, "expected second SetInputs to panic")
	out := catBufEnvInputs(t, &be)
	assert.Equal(t, "hello", out, "expected output")
}

func catBufEnvInputs(t *testing.T, be *xre.BufEnv) string {
	for in := range be.Inputs() {
		require.NoError(t, in.Err, "unexpected input error")
		_, err := be.DefaultOutput.ReadFrom(in.ReadCloser)
		require.NoError(t, err, "unexpected readfrom error")
		require.NoError(t, in.ReadCloser.Close(), "unexpected close error")
	}
	b, err := ioutil.ReadAll(&be.DefaultOutput)
	require.NoError(t, err, "unexpected readall error")
	return string(b)
}
