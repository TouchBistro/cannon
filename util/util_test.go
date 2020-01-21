package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

const fixtures = "../_fixtures"

func TestDirExists(t *testing.T) {
	path := fixtures + "/text_tests"
	assert.True(t, FileOrDirExists(path))
}

func TestFileExists(t *testing.T) {
	path := fixtures + "/text_tests/test.txt"
	assert.True(t, FileOrDirExists(path))
}

func TestFileNotExists(t *testing.T) {
	path := fixtures + "/notafile.txt"
	assert.False(t, FileOrDirExists(path))
}

func TestExecOutput(t *testing.T) {
	assert := assert.New(t)
	output, err := ExecOutput("echo", "Hello World")

	assert.NoError(err)
	assert.Equal("Hello World\n", output)
}

func TestExecOutputError(t *testing.T) {
	assert := assert.New(t)
	output, err := ExecOutput("notacmd")

	assert.Error(err)
	assert.Empty(output)
}

func TestExec(t *testing.T) {
	err := Exec("echo", ".", "Hello World")

	assert.NoError(t, err)
}

func TestExecError(t *testing.T) {
	err := Exec("notacmd", ".", "Hello World")

	assert.Error(t, err)
}
