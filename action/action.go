package action

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/TouchBistro/goutils/command"
	"github.com/TouchBistro/goutils/file"
	"github.com/pkg/errors"
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
	Run(t Target, args Arguments) (string, error)
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

// TODO(@cszatmary): The current config is super confusing. Try to make it better.

type Config struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Path   string `yaml:"path"`
	Run    string `yaml:"run"`
}

// Parse parses a config that describes an action and returns an Action.
func Parse(cfg Config) (Action, error) {
	if strings.HasSuffix(cfg.Type, "Text") || strings.HasSuffix(cfg.Type, "Line") {
		return parseTextAction(cfg)
	}
	switch cfg.Type {
	case fileCreate, fileReplace, fileCreateOrReplace, fileDelete:
		if cfg.Path == "" {
			return nil, errors.New("missing path for file action")
		}
		if cfg.Type == fileDelete {
			return fileAction{typ: cfg.Type, dst: cfg.Path}, nil
		}
		if cfg.Source == "" {
			return nil, errors.New("missing source for file action")
		}
		// Read and cache source file so we can reuse it for all targets
		data, err := os.ReadFile(cfg.Source)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read file %s", cfg.Source)
		}
		return fileAction{typ: cfg.Type, src: cfg.Source, dst: cfg.Path, data: data}, nil
	case "runCommand":
		const shellPrefix = "SHELL >> "
		var args []string
		if strings.HasPrefix(cfg.Run, shellPrefix) {
			shellCmd := strings.TrimPrefix(cfg.Run, shellPrefix)
			shellCmd = strings.TrimSpace(shellCmd)
			if shellCmd == "" {
				return nil, errors.New("missing shell command")
			}
			args = []string{"sh", "-c", shellCmd}
		} else {
			args = strings.Fields(cfg.Run)
			if len(args) == 0 {
				return nil, errors.New("missing args for run command")
			}
		}
		return commandAction{args: args, str: cfg.Run}, nil
	default:
		return nil, errors.Errorf("unsupported action type %s", cfg.Type)
	}
}

func parseTextAction(cfg Config) (Action, error) {
	// Path and Target are always required
	if cfg.Path == "" {
		return nil, errors.New("missing path for text action")
	}
	if cfg.Target == "" {
		return nil, errors.New("missing target for text action")
	}

	a := textAction{searchText: []byte(cfg.Target), path: cfg.Path}
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
		return nil, errors.Errorf("unsupported text action type %s", cfg.Type)
	}
	if cfg.Source == "" {
		return nil, errors.New("missing source for text action")
	}

	a.applyText = []byte(cfg.Source)
	return a, nil
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

func (a textAction) Run(t Target, args Arguments) (string, error) {
	// TODO(@cszatmary): Do we need to support vars in the search text?
	// Would be nice to not have that requirement, because then we could pre-compile the regex
	// in parse which would allow for catching errors earily.
	searchText, err := expandVars(a.searchText, args.Variables)
	if err != nil {
		return "", errors.Wrap(err, "failed to expand variables in action target")
	}
	applyText, err := expandVars(a.applyText, args.Variables)
	if err != nil {
		return "", errors.Wrap(err, "failed to expand variables in action source")
	}
	// Enable multi-line mode by adding flag if not a line action
	// https://golang.org/pkg/regexp/syntax/
	regexStr := string(searchText)
	if a.typ >= textReplace {
		regexStr = "(?m)" + regexStr
	}
	regex, err := regexp.Compile(regexStr)
	if err != nil {
		return "", errors.Wrap(err, "unable to compile regex from action target")
	}

	path := filepath.Join(t.Path(), a.path)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file %s", path)
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
		return "", errors.Wrapf(err, "failed to write file %s", path)
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

const (
	fileCreate          = "createFile"
	fileReplace         = "replaceFile"
	fileCreateOrReplace = "createOrReplaceFile"
	fileDelete          = "deleteFile"
)

// fileAction is an action that operates on files.
type fileAction struct {
	typ  string
	src  string
	dst  string // path in the target
	data []byte // src data; cached so it can be reused each run
}

func (a fileAction) Run(t Target, args Arguments) (string, error) {
	dstPath := filepath.Join(t.Path(), a.dst)
	exists := file.Exists(dstPath)
	switch a.typ {
	case fileCreate:
		if exists {
			return "", errors.Errorf("file %s already exists", dstPath)
		}
	case fileReplace, fileDelete:
		if !exists {
			return "", errors.Errorf("file %s does not exist", dstPath)
		}
	}

	if a.typ == fileDelete {
		if err := os.Remove(dstPath); err != nil {
			return "", errors.Wrapf(err, "failed to delete file %s", dstPath)
		}
		return fmt.Sprintf("Deleted file `%s`", a.dst), nil
	}

	data, err := expandVars(a.data, args.Variables)
	if err != nil {
		return "", errors.Wrapf(err, "failed to expand variables in file %s", a.src)
	}
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return "", errors.Wrapf(err, "failed to write file %s", dstPath)
	}
	if exists {
		return fmt.Sprintf("Replaced file `%s`", a.dst), nil
	}
	return fmt.Sprintf("Created file `%s`", a.dst), nil
}

func (a fileAction) String() string {
	switch a.typ {
	case fileCreate, fileReplace, fileCreateOrReplace:
		return fmt.Sprintf("%s: %q, source: %q", a.typ, a.dst, a.src)
	case fileDelete:
		return fmt.Sprintf("%s: %q", a.typ, a.dst)
	default:
		panic("impossible: invalid type: " + a.typ)
	}
}

// commandAction is an action that runs a command.
type commandAction struct {
	args []string // args[0] is the command, rest are args
	str  string   // the command string from the config; for printing
}

func (a commandAction) Run(t Target, _ Arguments) (string, error) {
	var errbuf bytes.Buffer
	cmd := command.New(command.WithStderr(&errbuf), command.WithDir(t.Path()))
	err := cmd.Exec(a.args[0], a.args[1:]...)
	if err != nil {
		return "", errors.Wrapf(err, "failed to run command %s at %s: %s", a.str, t.Path(), errbuf.String())
	}
	return fmt.Sprintf("Ran command `%s`", a.str), nil
}

func (a commandAction) String() string {
	return fmt.Sprintf("run: %s", a.str)
}

// Regex to match variable substitution of the form ${VAR}
var varRegex = regexp.MustCompile(`\$\{([\w-@:]+)\}`)

// expandVars returns a copy of src with variables of the form ${VAR} expanded.
// If src contains no vars, it is returned unchanged. If a variable value is not
// found, an error will be returned.
func expandVars(src []byte, vars map[string]string) ([]byte, error) {
	matches := varRegex.FindAllSubmatchIndex(src, -1)
	if matches == nil {
		return src, nil
	}

	lastEndIndex := 0
	var b []byte
	for _, match := range matches {
		// match[0] is the start index of the whole match
		startIndex := match[0]
		// match[1] is the end index of the whole match (exclusive)
		endIndex := match[1]
		// match[2] is start index of group
		startIndexGroup := match[2]
		// match[3] is end index of group (exclusive)
		endIndexGroup := match[3]

		varName := string(src[startIndexGroup:endIndexGroup])
		varValue, ok := vars[varName]
		if !ok {
			return nil, errors.Errorf("unknown variable %q", varName)
		}

		b = append(b, src[lastEndIndex:startIndex]...)
		b = append(b, varValue...)
		lastEndIndex = endIndex
	}
	b = append(b, src[lastEndIndex:]...)
	return b, nil
}
