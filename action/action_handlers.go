package action

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
)

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

func replaceLine(action Action, targetRegex *regexp.Regexp, fileData []byte) ([]byte, string) {
	lines := strings.Split(string(fileData), "\n")

	for i, line := range lines {
		if targetRegex.MatchString(line) {
			lines[i] = action.Source
		}
	}

	output := strings.Join(lines, "\n")
	msg := fmt.Sprintf("Replaced line `%s` with `%s` in `%s`", action.Target, action.Source, action.Path)
	return []byte(output), msg
}

func deleteLine(action Action, targetRegex *regexp.Regexp, fileData []byte) ([]byte, string) {
	lines := strings.Split(string(fileData), "\n")
	filteredLines := make([]string, 0)

	// Filter all lines that match the line to delete
	for _, line := range lines {
		if !targetRegex.MatchString(line) {
			filteredLines = append(filteredLines, line)
		}
	}

	output := strings.Join(filteredLines, "\n")
	msg := fmt.Sprintf("Deleted line `%s` in `%s`\n", action.Target, action.Path)
	return []byte(output), msg
}

func replaceText(action Action, targetRegex *regexp.Regexp, fileData []byte) ([]byte, string) {
	contents := targetRegex.ReplaceAllString(string(fileData), action.Source)

	msg := fmt.Sprintf("Replaced text `%s` with `%s` in `%s`", action.Target, action.Source, action.Path)
	return []byte(contents), msg
}

func appendText(action Action, targetRegex *regexp.Regexp, fileData []byte) ([]byte, string) {
	contents := targetRegex.ReplaceAllStringFunc(string(fileData), func(target string) string {
		return target + action.Source
	})

	msg := fmt.Sprintf("Appended text `%s` to all occurrences of `%s` in `%s`", action.Source, action.Target, action.Path)
	return []byte(contents), msg
}

func deleteText(action Action, targetRegex *regexp.Regexp, fileData []byte) ([]byte, string) {
	// Get a slice of all substrings that don't match regex
	components := targetRegex.Split(string(fileData), -1)
	contents := strings.Join(components, "")

	msg := fmt.Sprintf("Deleted all occurrences of `%s` in `%s`\n", action.Target, action.Path)
	return []byte(contents), msg
}

func createFile(action Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := filepath.Join(cwd, action.Source)

	filePath := filepath.Join(repoPath, action.Path)
	if file.FileOrDirExists(filePath) {
		return "", errors.Errorf("File at path %s already exists", filePath)
	}

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create file at %s", filePath)
	}

	return fmt.Sprintf("Created file `%s`", action.Path), nil
}

func deleteFile(action Action, repoPath string) (string, error) {
	filePath := filepath.Join(repoPath, action.Path)
	if !file.FileOrDirExists(filePath) {
		return "", errors.Errorf("File at path %s does not exist", filePath)
	}

	err := os.Remove(filePath)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete file %s", filePath)
	}

	return fmt.Sprintf("Deleted file `%s`", action.Path), nil
}

func replaceFile(action Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := filepath.Join(cwd, action.Source)

	filePath := filepath.Join(repoPath, action.Path)
	if !file.FileOrDirExists(filePath) {
		return "", errors.Errorf("File at path %s does not exist", filePath)
	}

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to replace file at %s", filePath)
	}

	return fmt.Sprintf("Replaced file `%s`", action.Path), nil
}

func createOrReplaceFile(action Action, repoPath, repoName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "unable to get current working directory")
	}
	sourceFilePath := filepath.Join(cwd, action.Source)

	filePath := filepath.Join(repoPath, action.Path)

	err = copyFile(sourceFilePath, filePath, repoName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create or replace file at %s", filePath)
	}

	return fmt.Sprintf("Created or replaced file `%s`", action.Path), nil
}

func runCommand(action Action, repoPath string) (string, error) {
	const shellPrefix = "SHELL >> "
	var cmdName string
	var args []string

	if strings.HasPrefix(action.Run, shellPrefix) {
		cmdName = "sh"
		shellCmd := strings.TrimPrefix(action.Run, shellPrefix)
		args = []string{"-c", shellCmd}
	} else {
		fields := strings.Fields(action.Run)
		cmdName = fields[0]
		args = fields[1:]
	}

	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = repoPath

	err := cmd.Run()
	if err != nil {
		return "", errors.Wrapf(err, "failed to run command %s at %s", action.Run, repoPath)
	}

	return fmt.Sprintf("Ran command `%s`", action.Run), nil
}
