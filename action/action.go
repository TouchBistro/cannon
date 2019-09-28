package action

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/TouchBistro/cannon/config"
	"github.com/TouchBistro/cannon/util"
	"github.com/pkg/errors"
)

func expandRepoVar(source, repoName string) string {
	return strings.ReplaceAll(source, "$REPONAME", repoName)
}

func copyFile(fromPath, toPath, repoName string) error {
	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(fromPath)
	if err != nil {
		return errors.Wrapf(err, "failed to read file %s", fromPath)
	}

	contents := expandRepoVar(string(data), repoName)
	err = ioutil.WriteFile(toPath, []byte(contents), 0644)
	return errors.Wrapf(err, "failed to write file %s", toPath)
}

func ReplaceLine(action config.Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
	}

	sourceStr := expandRepoVar(action.Source, repoName)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if line == action.Target {
			lines[i] = sourceStr
		}
	}

	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return fmt.Sprintf("Replaced line `%s` with `%s` in `%s`", action.Target, sourceStr, action.Path), nil
}

func DeleteLine(action config.Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
	}

	lines := strings.Split(string(data), "\n")
	filteredLines := make([]string, 0)
	targetStr := expandRepoVar(action.Target, repoName)

	// Filter all lines that match the line to delete
	for _, line := range lines {
		if line != targetStr {
			filteredLines = append(filteredLines, line)
		}
	}

	output := strings.Join(filteredLines, "\n")
	err = ioutil.WriteFile(filePath, []byte(output), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return fmt.Sprintf("Deleted line `%s` in `%s`\n", targetStr, action.Path), nil
}

func ReplaceText(action config.Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	// Do lazy way for now, can optimize later if needed
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
	}

	sourceStr := expandRepoVar(action.Source, repoName)
	contents := strings.ReplaceAll(string(data), action.Target, sourceStr)

	err = ioutil.WriteFile(filePath, []byte(contents), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return fmt.Sprintf("Replaced text `%s` with `%s` in `%s`", action.Target, sourceStr, action.Path), nil
}

func AppendText(action config.Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
	}

	regex, err := regexp.Compile(action.Target)
	if err != nil {
		return "", errors.Wrap(err, "unable to compile regex from action target")
	}

	sourceStr := expandRepoVar(action.Source, repoName)
	contents := regex.ReplaceAllStringFunc(string(data), func(target string) string {
		return target + sourceStr
	})

	err = ioutil.WriteFile(filePath, []byte(contents), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return fmt.Sprintf("Appended text `%s` to all occurrences of `%s` in `%s`", sourceStr, action.Target, action.Path), nil
}

func DeleteText(action config.Action, repoPath, repoName string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", filePath)
	}

	targetStr := expandRepoVar(action.Target, repoName)
	regex, err := regexp.Compile(targetStr)
	if err != nil {
		return "", errors.Wrap(err, "unable to compile regex from action target")
	}

	// Get a slice of all substrings that don't match regex
	components := regex.Split(string(data), -1)
	contents := strings.Join(components, "")

	err = ioutil.WriteFile(filePath, []byte(contents), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", filePath)
	}

	return fmt.Sprintf("Deleted all occurrences of `%s` in `%s`\n", targetStr, action.Path), nil
}

func CreateFile(action config.Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if util.FileOrDirExists(filePath) {
		return "", errors.New(fmt.Sprintf("File at path %s already exists", filePath))
	}

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file at %s", filePath)
	}

	return fmt.Sprintf("Created file `%s`", action.Path), nil
}

func DeleteFile(action config.Action, repoPath string) (string, error) {
	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if !util.FileOrDirExists(filePath) {
		return "", errors.New(fmt.Sprintf("File at path %s does not exist", filePath))
	}

	err := os.Remove(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete file %s", filePath)
	}

	return fmt.Sprintf("Deleted file `%s`", action.Path), nil
}

func ReplaceFile(action config.Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)
	if !util.FileOrDirExists(filePath) {
		return "", errors.New(fmt.Sprintf("File at path %s does not exist", filePath))
	}

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to replace file at %s", filePath)
	}

	return fmt.Sprintf("Replaced file `%s`", action.Path), nil
}

func CreateOrReplaceFile(action config.Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := fmt.Sprintf("%s/%s", cwd, action.Source)

	filePath := fmt.Sprintf("%s/%s", repoPath, action.Path)

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create or replace file at %s", filePath)
	}

	return fmt.Sprintf("Created or replaced file `%s`", action.Path), nil
}

func RunCommand(action config.Action, repoPath string) (string, error) {
	args := strings.Fields(action.Run)

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath

	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "failed to run command %s at %s", action.Run, repoPath)
	}

	return fmt.Sprintf("Ran command `%s`", action.Run), nil
}
