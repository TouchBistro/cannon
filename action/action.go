package action

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

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

func executeTextAction(action Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	fileData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
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
		return "", errors.New(fmt.Sprintf("invalid action type %s", action.Type))
	}

	outputData, msg := actionFn(action, regex, fileData)

	err = ioutil.WriteFile(filePath, []byte(outputData), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return msg, nil
}

func ExecuteAction(action Action, repoPath, repoName string) (string, error) {
	if action.IsLineAction() || action.IsTextAction() {
		msg, err := executeTextAction(action, repoPath, repoName)
		if err != nil {
			return "", errors.Wrapf(err, "failed to execute text action %s", action.Type)
		}

		return msg, err
	}

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
		return "", errors.New(fmt.Sprintf("invalid action type %s", action.Type))
	}
}
