package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/file"
	"github.com/TouchBistro/goutils/text"
)

// Action represents an action that can be applied.
//
// The Run method applies the action to a given target
// and returns a string describing the result.
//
// Actions should not mutate themselves or have side effects since they are intended
// to be run multiple times with different targets, possible on different goroutines.
//
// The String method provides a string description of the action.
type Action interface {
	Run(ctx context.Context, t Target, args Arguments) (string, error)
	String() string
}

// Target represents an item that an Action will be applied to.
type Target interface {
	Path() string
}

// Arguments contains additional arguments that can be provided when running an Action.
type Arguments struct {
	// Values for variables that can be expanded during actions.
	Variables map[string]string
}

// Config is used to configure an action.
// It is passed to Parse to create an Action instance.
type Config struct {
	// Identifies the type of action. Required for all actions.
	Type string `yaml:"type"`

	// The text to search for in a text action.
	SearchText string `yaml:"searchText"`
	// The text to apply in a text action.
	ApplyText string `yaml:"applyText"`
	// The path to the file in a text file.
	// Must be relative to the target root.
	Path string `yaml:"path"`

	// The source file to use in a file action.
	SrcPath string `yaml:"srcPath"`
	// The destination file to use in a file action.
	// Must be relative to the target root.
	DstPath string `yaml:"dstPath"`

	// The command to run in a command action.
	Run string `yaml:"run"`
}

// Parse parses a config that describes an action and returns an Action.
func Parse(cfg Config) (Action, error) {
	switch {
	case strings.HasSuffix(cfg.Type, "Text") || strings.HasSuffix(cfg.Type, "Line"):
		return parseTextAction(cfg)
	case strings.HasSuffix(cfg.Type, "File"):
		return parseFileAction(cfg)
	case strings.HasSuffix(cfg.Type, "Command"):
		return parseCommandAction(cfg)
	default:
		return nil, fmt.Errorf("unsupported action type %s", cfg.Type)
	}
}

func parseTextAction(cfg Config) (Action, error) {
	// Path and Target are always required
	if cfg.Path == "" {
		return nil, errors.New("missing path for text action")
	}
	if cfg.SearchText == "" {
		return nil, errors.New("missing search text for text action")
	}

	a := textAction{searchText: []byte(cfg.SearchText), path: cfg.Path}
	switch cfg.Type {
	case "replaceLine":
		a.typ = textReplaceLine
	case "deleteLine":
		a.typ = textDeleteLine
		return a, nil
	case "replaceText":
		a.typ = textReplace
	case "appendText":
		a.typ = textAppend
	case "deleteText":
		a.typ = textDelete
		return a, nil
	default:
		return nil, fmt.Errorf("unsupported text action type %s", cfg.Type)
	}
	if cfg.ApplyText == "" {
		return nil, errors.New("missing apply text for text action")
	}

	a.applyText = []byte(cfg.ApplyText)
	return a, nil
}

func parseFileAction(cfg Config) (Action, error) {
	if cfg.DstPath == "" {
		return nil, errors.New("missing destination path for file action")
	}

	a := fileAction{dst: cfg.DstPath}
	switch cfg.Type {
	case "createFile":
		a.typ = fileCreate
	case "replaceFile":
		a.typ = fileReplace
	case "createOrReplaceFile":
		a.typ = fileCreateOrReplace
	case "deleteFile":
		a.typ = fileDelete
		return a, nil
	default:
		return nil, fmt.Errorf("unsupported file action type %s", cfg.Type)
	}
	if cfg.SrcPath == "" {
		return nil, errors.New("missing source path for file action")
	}

	// Read and cache source file so we can reuse it for all targets
	data, err := os.ReadFile(cfg.SrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", cfg.SrcPath, err)
	}
	a.src = cfg.SrcPath
	a.data = data
	return a, nil
}

func parseCommandAction(cfg Config) (Action, error) {
	if cfg.Run == "" {
		return nil, errors.New("missing run field for command action")
	}

	var args []string
	switch cfg.Type {
	case "runCommand":
		args = strings.Fields(cfg.Run)
	case "shellCommand":
		args = []string{"sh", "-c", cfg.Run}
	default:
		return nil, fmt.Errorf("unsupported command action type %s", cfg.Type)
	}
	return commandAction{args: args, str: cfg.Run}, nil
}

type textActionType int

const (
	textReplaceLine textActionType = iota
	textDeleteLine
	textReplace
	textAppend
	textDelete
)

// textAction is an action that makes changes to the text in a file.
type textAction struct {
	typ        textActionType
	searchText []byte // text that will be matched; it's a regex
	applyText  []byte // text that will be applied in non-delete types
	path       string
}

func (a textAction) Run(_ context.Context, t Target, args Arguments) (string, error) {
	vm := text.NewVariableMapper(args.Variables)
	searchText := text.ExpandVariables(a.searchText, vm.Map)
	if len(vm.Missing()) > 0 {
		return "", fmt.Errorf("failed to expand variables in action target, unknown variables %q", strings.Join(vm.Missing(), ", "))
	}
	applyText := text.ExpandVariables(a.applyText, vm.Map)
	if len(vm.Missing()) > 0 {
		return "", fmt.Errorf("failed to expand variables in action source, unknown variables %q", strings.Join(vm.Missing(), ", "))
	}
	// Enable multi-line mode by adding flag if not a line action
	// https://golang.org/pkg/regexp/syntax/
	regexStr := string(searchText)
	if a.typ >= textReplace {
		regexStr = "(?m)" + regexStr
	}
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return "", fmt.Errorf("unable to compile regex from action target: %w", err)
	}

	path := filepath.Join(t.Path(), a.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var output []byte
	var msg string
	switch a.typ {
	case textReplaceLine:
		lines := bytes.Split(data, []byte{'\n'})
		for i, line := range lines {
			if regex.Match(line) {
				lines[i] = applyText
			}
		}
		output = bytes.Join(lines, []byte{'\n'})
		msg = fmt.Sprintf("Replaced line `%s` with `%s` in `%s`", searchText, applyText, a.path)
	case textDeleteLine:
		lines := bytes.Split(data, []byte{'\n'})
		var filtered [][]byte
		// Filter all lines that match the line to delete
		for _, line := range lines {
			if !regex.Match(line) {
				filtered = append(filtered, line)
			}
		}
		output = bytes.Join(filtered, []byte{'\n'})
		msg = fmt.Sprintf("Deleted line `%s` in `%s`", searchText, a.path)
	case textReplace:
		output = regex.ReplaceAll(data, applyText)
		msg = fmt.Sprintf("Replaced text `%s` with `%s` in `%s`", searchText, applyText, a.path)
	case textAppend:
		output = regex.ReplaceAllFunc(data, func(m []byte) []byte {
			// Make sure we copy m and don't mutate it since it is a slice of data
			out := append([]byte{}, m...)
			return append(out, applyText...)
		})
		msg = fmt.Sprintf("Appended text `%s` to all occurrences of `%s` in `%s`", applyText, searchText, a.path)
	case textDelete:
		// Get a slice of all substrings that don't match regex
		// TODO(@cszatmary): Could optimize this by doing the split ourselves so we don't
		// need to convert it to a string first.
		parts := regex.Split(string(data), -1)
		for _, p := range parts {
			output = append(output, p...)
		}
		msg = fmt.Sprintf("Deleted all occurrences of `%s` in `%s`", searchText, a.path)
	default:
		panic("impossible: invalid type")
	}

	if err := os.WriteFile(path, output, 0o644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return msg, nil
}

func (a textAction) String() string {
	switch a.typ {
	case textReplaceLine:
		return fmt.Sprintf("replace line: %q\n  with: %q\n  path: %q", a.searchText, a.applyText, a.path)
	case textDeleteLine:
		return fmt.Sprintf("delete line: %q\n  path: %q", a.searchText, a.path)
	case textReplace:
		return fmt.Sprintf("replace text: %q\n  with: %q\n  path: %q", a.searchText, a.applyText, a.path)
	case textAppend:
		return fmt.Sprintf("append text: %q\n  to: %q\n  path: %q", a.applyText, a.searchText, a.path)
	case textDelete:
		return fmt.Sprintf("delete text: %q\n  path: %q", a.searchText, a.path)
	default:
		panic("impossible: invalid type")
	}
}

type fileActionType int

const (
	fileCreate fileActionType = iota
	fileReplace
	fileCreateOrReplace
	fileDelete
)

// fileAction is an action that operates on files.
type fileAction struct {
	typ  fileActionType
	src  string
	dst  string // path in the target
	data []byte // src data; cached so it can be reused each run
}

func (a fileAction) Run(_ context.Context, t Target, args Arguments) (string, error) {
	dstPath := filepath.Join(t.Path(), a.dst)
	exists := file.Exists(dstPath)
	switch a.typ {
	case fileCreate:
		if exists {
			return "", fmt.Errorf("file %s already exists", dstPath)
		}
	case fileReplace, fileDelete:
		if !exists {
			return "", fmt.Errorf("file %s does not exist", dstPath)
		}
	}

	if a.typ == fileDelete {
		if err := os.Remove(dstPath); err != nil {
			return "", fmt.Errorf("failed to delete file %s: %w", dstPath, err)
		}
		return fmt.Sprintf("Deleted file `%s`", a.dst), nil
	}

	vm := text.NewVariableMapper(args.Variables)
	data := text.ExpandVariables(a.data, vm.Map)
	if len(vm.Missing()) > 0 {
		return "", fmt.Errorf("failed to expand variables in file %s, unknown variables %q", a.src, strings.Join(vm.Missing(), ", "))
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", dstPath, err)
	}
	if exists {
		return fmt.Sprintf("Replaced file `%s`", a.dst), nil
	}
	return fmt.Sprintf("Created file `%s`", a.dst), nil
}

func (a fileAction) String() string {
	switch a.typ {
	case fileCreate:
		return fmt.Sprintf("create file: %q\n  from: %q", a.dst, a.src)
	case fileReplace:
		return fmt.Sprintf("replace file: %q\n  with: %q", a.dst, a.src)
	case fileCreateOrReplace:
		return fmt.Sprintf("create or replace file: %q\n  with: %q", a.dst, a.src)
	case fileDelete:
		return fmt.Sprintf("delete file: %q", a.dst)
	default:
		panic("impossible: invalid type")
	}
}

// commandAction is an action that runs a command.
type commandAction struct {
	args []string // args[0] is the command, rest are args
	str  string   // the command string from the config; for printing
}

func (a commandAction) Run(ctx context.Context, t Target, _ Arguments) (string, error) {
	var errbuf bytes.Buffer
	cmd := exec.CommandContext(ctx, a.args[0], a.args[1:]...)
	cmd.Stderr = &errbuf
	cmd.Dir = t.Path()
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run command %s at %s: %s: %w", a.str, t.Path(), errbuf.String(), err)
	}
	return fmt.Sprintf("Ran command `%s`", a.str), nil
}

func (a commandAction) String() string {
	return fmt.Sprintf("run: %s", a.str)
}
