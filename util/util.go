package util

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func CopyFile(fromPath, toPath string) error {
	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(fromPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", fromPath)
	}

	err = ioutil.WriteFile(toPath, data, 0644)
	return errors.Wrapf(err, "failed to write file %s", toPath)
}

func ReadYaml(path string, val interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrapf(err, "failed to open file %s", path)
	}
	defer file.Close()

	dec := yaml.NewDecoder(file)
	err = dec.Decode(val)
	return errors.Wrapf(err, "failed to decode yaml file %s", path)
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
