package util

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func FileOrDirExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
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
