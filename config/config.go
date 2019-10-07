package config

import (
	"fmt"
	"os"

	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
)

type Action struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Path   string `yaml:"path"`
	Run    string `yaml:"run"`
}

type Repo struct {
	Name string `yaml:"name"`
	Base string `yaml:"base"`
}

type CannonConfig struct {
	Repos   []Repo   `yaml:"repos"`
	Actions []Action `yaml:"actions"`
}

const (
	ActionReplaceLine         = "replaceLine"
	ActionReplaceText         = "replaceText"
	ActionAppendText          = "appendText"
	ActionCreateFile          = "createFile"
	ActionDeleteFile          = "deleteFile"
	ActionReplaceFile         = "replaceFile"
	ActionCreateOrReplaceFile = "createOrReplaceFile"
	ActionRunCommand          = "runCommand"
)

var (
	config    CannonConfig
	cannonDir string
)

func Init(path string) error {
	cannonDir = fmt.Sprintf("%s/.cannon", os.Getenv("HOME"))

	// Create ~/.cannon directory if it doesn't exist
	if !util.FileOrDirExists(cannonDir) {
		err := os.Mkdir(cannonDir, 0755)
		if err != nil {
			return errors.Wrapf(err, "failed to create cannon directory at %s", cannonDir)
		}
	}

	if !util.FileOrDirExists(path) {
		return errors.Errorf("No such file %s", path)
	}

	err := util.ReadYaml(path, &config)
	return errors.Wrapf(err, "couldn't read yaml file at %s", path)
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
