package config

import (
	"io"
	"os"
	"path/filepath"

	"github.com/TouchBistro/cannon/action"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Repo struct {
	Name string `yaml:"name"`
	Base string `yaml:"base"`
}

func (repo Repo) BaseBranch() string {
	if repo.Base == "" {
		return "master"
	}
	return repo.Base
}

type CannonConfig struct {
	Repos   []Repo          `yaml:"repos"`
	Actions []action.Action `yaml:"actions"`
}

var (
	config    CannonConfig
	cannonDir string
)

func Init(r io.Reader) error {
	hd, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "failed to get user's home directory")
	}
	cannonDir = filepath.Join(hd, ".cannon")

	// Create ~/.cannon directory if it doesn't exist
	if err := os.MkdirAll(cannonDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to create cannon directory at %s", cannonDir)
	}

	err = yaml.NewDecoder(r).Decode(&config)
	return errors.Wrap(err, "couldn't read yaml config file")
}

func Config() *CannonConfig {
	return &config
}

func CannonDir() string {
	return cannonDir
}
