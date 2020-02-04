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

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func ExecOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	return stdout.String(), errors.Wrapf(err, "exec failed for command %s: %s", name, stderr.String())
}

func Exec(name, dir string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir

	err := cmd.Run()
	if err != nil {
		return errors.Wrapf(err, "Exec failed to run %s %s", name, arg)
	}

	return nil
}
