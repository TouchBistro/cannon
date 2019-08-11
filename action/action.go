package action

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
)

func ReplaceLine(action config.Action, repoPath string) error {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", filePath)
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if line == action.Target {
			lines[i] = action.Source
		}
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", filePath)
	}

	fmt.Printf("Replaced line '%s' with '%s' in %s\n", action.Target, action.Source, filePath)
	return nil
}

func ReplaceText(action config.Action, repoPath string) error {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", filePath)
	}

	contents := strings.ReplaceAll(string(data), action.Target, action.Source)

	err = ioutil.WriteFile(filePath, []byte(contents), 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", filePath)
	}

	fmt.Printf("Replaced text '%s' with '%s' in %s\n", action.Target, action.Source, filePath)
	return nil
}

func CreateFile(action config.Action, repoPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if util.FileOrDirExists(filePath) {
		return errors.New(fmt.Sprintf("File at path %s already exists", filePath))
	}

	err = util.CopyFile(sourceFilePath, filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file at %s", filePath)
	}

	fmt.Printf("Created file %s from %s\n", filePath, sourceFilePath)
	return nil
}

func DeleteFile(action config.Action, repoPath string) error {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if !util.FileOrDirExists(filePath) {
		return errors.New(fmt.Sprintf("File at path %s does not exist", filePath))
	}

	err := os.Remove(filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to delete file %s", filePath)
	}

	fmt.Printf("Deleted file %s\n", filePath)
	return nil
}

func ReplaceFile(action config.Action, repoPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if !util.FileOrDirExists(filePath) {
		return errors.New(fmt.Sprintf("File at path %s does not exist", filePath))
	}

	err = util.CopyFile(sourceFilePath, filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to replace file at %s", filePath)
	}

	fmt.Printf("Replaced file %s with %s\n", filePath, sourceFilePath)
	return nil
}

func CreateOrReplaceFile(action config.Action, repoPath string) error {
	cwd, err := os.Getwd()
	if err != nil {
		errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	err = util.CopyFile(sourceFilePath, filePath)
	if err != nil {
		return errors.Wrapf(err, "failed to create or replace file at %s", filePath)
	}

	fmt.Printf("Created or replaced file %s with %s\n", filePath, sourceFilePath)
	return nil
}
