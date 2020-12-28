package action

import (
	"fmt"
	"io"
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

func (a Action) IsLineAction() bool {
	return strings.HasSuffix(a.Type, "Line")
}

func (a Action) IsTextAction() bool {
	return strings.HasSuffix(a.Type, "Text")
}

type TruncateWriterAt interface {
	io.WriterAt
	Truncate(size int64) error
}

func ExecuteTextAction(action Action, r io.Reader, w TruncateWriterAt, repoName string) (string, error) {
	// Do lazy way for now, can optimize later if needed
	fileData, err := ioutil.ReadAll(r)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", action.Path)
	}

	parts := strings.Split(repoName, "/")
	vars := map[string]string{
		"REPO_OWNER": parts[0],
		"REPO_NAME":  parts[1],
	}
	action.Source, err = expandVars(action.Source, vars)
	if err != nil {
		return "", errors.Wrap(err, "failed to expand variables in action source")
	}
	action.Target, err = expandVars(action.Target, vars)
	if err != nil {
		return "", errors.Wrap(err, "failed to expand variables in action target")
	}

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

	err = w.Truncate(0)
	if err != nil {
		return "", errors.Wrapf(err, "failed to truncate file %s", action.Path)
	}

	_, err = w.WriteAt(outputData, 0)
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

func expandVars(str string, vars map[string]string) (string, error) {
	// Regex to match variable substitution of the form ${VAR}
	regex := regexp.MustCompile(`\$\{([\w-@:]+)\}`)
	var result string

	lastEndIndex := 0
	for _, match := range regex.FindAllStringSubmatchIndex(str, -1) {
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := str[startIndexGroup:endIndexGroup]
		varValue, ok := vars[varName]
		if !ok {
			return "", fmt.Errorf("unknown variable %q", varName)
		}

		result += str[lastEndIndex:startIndex]
		result += varValue
		lastEndIndex = endIndex
	}

	result += str[lastEndIndex:]
	return result, nil
}
