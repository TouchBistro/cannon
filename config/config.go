package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Repo struct {
	Name string `yaml:"name"`
	Base string `yaml:"base"`
}

type CannonConfig struct {
	Repos   []Repo          `yaml:"repos"`
	Actions []action.Action `yaml:"actions"`
}

var (
	config    CannonConfig
	cannonDir string
)

func Init(configReader io.Reader) error {
	cannonDir = filepath.Join(os.Getenv("HOME"), ".cannon")

	// Create ~/.cannon directory if it doesn't exist
	if !util.FileOrDirExists(cannonDir) {
		err := os.Mkdir(cannonDir, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create cannon directory at %s", cannonDir)
		}
	}

	dec := yaml.NewDecoder(configReader)
	err := dec.Decode(&config)
	return errors.Wrap(err, "couldn't read yaml config file")
}

func Config() *CannonConfig {
	return &config
}

func CannonDir() string {
	return cannonDir
}

func (repo Repo) BaseBranch() string {
	if repo.Base == "" {
		return "master"
	}

	return repo.Base
}
