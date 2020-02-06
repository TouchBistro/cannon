package action

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
)

const (
	ActionReplaceLine         = "replaceLine"
	ActionDeleteLine          = "deleteLine"
	ActionReplaceText         = "replaceText"
	ActionAppendText          = "appendText"
	ActionDeleteText          = "deleteText"
	ActionCreateFile          = "createFile"
	ActionDeleteFile          = "deleteFile"
	ActionReplaceFile         = "replaceFile"
	ActionCreateOrReplaceFile = "createOrReplaceFile"
	ActionRunCommand          = "runCommand"
)

func (a Action) String() string {
	switch a.Type {
	case ActionReplaceLine, ActionReplaceText, ActionAppendText:
		return fmt.Sprintf("- type: %s\n  source: %s\n  target: %s\n  path: %s", a.Type, a.Source, a.Target, a.Path)
	case ActionDeleteLine, ActionDeleteText:
		return fmt.Sprintf("- type: %s\n  target: %s\n  path: %s", a.Type, a.Target, a.Path)
	case ActionCreateFile, ActionReplaceFile, ActionCreateOrReplaceFile:
		return fmt.Sprintf("- type: %s\n  source: %s\n  path: %s", a.Type, a.Source, a.Path)
	case ActionDeleteFile:
		return fmt.Sprintf("- type: %s\n  path: %s", a.Type, a.Path)
	case ActionRunCommand:
		return fmt.Sprintf("- type: %s\n  run: %s", a.Type, a.Run)
	default:
		return fmt.Sprintf("Unsupported type: %s", a.Type)
	}
}

type Action struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Path   string `yaml:"path"`
	Run    string `yaml:"run"`
}

func (a Action) IsLineAction() bool {
	return strings.HasSuffix(a.Type, "Line")
}

func (a Action) IsTextAction() bool {
	return strings.HasSuffix(a.Type, "Text")
}

func expandRepoVar(source, repoName string) string {
	return strings.ReplaceAll(source, "$REPONAME", repoName)
}

func ExecuteTextAction(action Action, r io.Reader, w util.OffsetWriter, repoName string) (string, error) {
	// Do lazy way for now, can optimize later if needed
	fileData, err := ioutil.ReadAll(r)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", action.Path)
	}

	action.Source = expandRepoVar(action.Source, repoName)
	action.Target = expandRepoVar(action.Target, repoName)

	// Enable multi-line mode by adding flag if text action
	// https://golang.org/pkg/regexp/syntax/
	regexStr := action.Target
	if action.IsTextAction() {
		regexStr = "(?m)" + regexStr
	}

	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return "", errors.Wrap(err, "unable to compile regex from action target")
	}

	var actionFn func(Action, *regexp.Regexp, []byte) ([]byte, string)

	switch action.Type {
	case ActionReplaceLine:
		actionFn = replaceLine
	case ActionDeleteLine:
		actionFn = deleteLine
	case ActionReplaceText:
		actionFn = replaceText
	case ActionAppendText:
		actionFn = appendText
	case ActionDeleteText:
		actionFn = deleteText
	default:
		return "", errors.Errorf("invalid action type %s", action.Type)
	}

	outputData, msg := actionFn(action, regex, fileData)

	_, err = w.WriteAt([]byte(outputData), 0)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", action.Path)
	}

	return msg, nil
}

func ExecuteFileAction(action Action, repoPath, repoName string) (string, error) {
	switch action.Type {
	case ActionCreateFile:
		return createFile(action, repoPath, repoName)
	case ActionDeleteFile:
		return deleteFile(action, repoPath)
	case ActionReplaceFile:
		return replaceFile(action, repoPath, repoName)
	case ActionCreateOrReplaceFile:
		return createOrReplaceFile(action, repoPath, repoName)
	case ActionRunCommand:
		return runCommand(action, repoPath)
	default:
		return "", errors.Errorf("invalid action type %s", action.Type)
	}
}
