package config

import (
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
)

type Action struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Path   string `yaml:"path"`
}

type CannonConfig struct {
	Repos   []string `yaml:"repos"`
	Actions []Action `yaml:"actions"`
}

const (
	ActionReplaceLine         = "replaceLine"
	ActionReplaceText         = "replaceText"
	ActionCreateFile          = "createFile"
	ActionDeleteFile          = "deleteFile"
	ActionReplaceFile         = "replaceFile"
	ActionCreateOrReplaceFile = "createOrReplaceFile"
)

var config CannonConfig

func Init(path string) error {
	if !util.FileOrDirExists(path) {
		return errors.Errorf("No such file %s", path)
	}

	err := util.ReadYaml(path, &config)
	return errors.Wrapf(err, "couldn't read yaml file at %s", path)
}

func Config() *CannonConfig {
	return &config
}
