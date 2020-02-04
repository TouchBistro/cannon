package util

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

type OffsetWriter interface {
	WriteAt(b []byte, off int64) (n int, err error)
}

// FileOrDirExists checks if the file or directory at a given path exists.
func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

// ExecOutput executes a shell command and returns stdout as a string.
func ExecOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return stdout.String(), errors.Wrapf(err, "exec failed for command %s: %s", name, stderr.String())
}

// Exec executes a shell command and returns an error if it failed.
func Exec(name, dir string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Exec failed to run %s %s", name, arg)
	}

	return nil
}
