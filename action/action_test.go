package action_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/goutils/file"
)

type pathTarget string

func (p pathTarget) Path() string {
	return string(p)
}

const inputText = `# HYPE ZONE
This file is ***hype***.

## Hype Section
This section is pretty hype.
`

func TestTextAction(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		cfg     action.Config
		vars    map[string]string
		wantMsg string
		out     string
	}{
		{
			name: "replace line",
			in:   inputText,
			cfg: action.Config{
				Type:       "replaceLine",
				ApplyText:  "# WOKE ZONE",
				SearchText: "# HYPE ZONE",
				Path:       "replace_line.md",
			},
			wantMsg: "Replaced line `# HYPE ZONE` with `# WOKE ZONE` in `replace_line.md`",
			out: `# WOKE ZONE
This file is ***hype***.

## Hype Section
This section is pretty hype.
`,
		},
		{
			name: "delete line",
			in:   inputText,
			cfg: action.Config{
				Type:       "deleteLine",
				SearchText: "## Hype Section",
				Path:       "delete_line.md",
			},
			wantMsg: "Deleted line `## Hype Section` in `delete_line.md`",
			out: `# HYPE ZONE
This file is ***hype***.

This section is pretty hype.
`,
		},
		{
			name: "replace text",
			in:   inputText,
			cfg: action.Config{
				Type:       "replaceText",
				ApplyText:  "*****",
				SearchText: "^#.+",
				Path:       "replace_text.md",
			},
			wantMsg: "Replaced text `^#.+` with `*****` in `replace_text.md`",
			out: `*****
This file is ***hype***.

*****
This section is pretty hype.
`,
		},
		{
			name: "append text",
			in:   inputText,
			cfg: action.Config{
				Type:       "appendText",
				ApplyText:  " --- ${REPO_OWNER} - ${REPO_NAME}",
				SearchText: "^#.+",
				Path:       "append_text.md",
			},
			vars: map[string]string{
				"REPO_OWNER": "TouchBistro",
				"REPO_NAME":  "node-boilerplate",
			},
			wantMsg: "Appended text ` --- TouchBistro - node-boilerplate` to all occurrences of `^#.+` in `append_text.md`",
			out: `# HYPE ZONE --- TouchBistro - node-boilerplate
This file is ***hype***.

## Hype Section --- TouchBistro - node-boilerplate
This section is pretty hype.
`,
		},
		{
			name: "delete text",
			in:   inputText,
			cfg: action.Config{
				Type:       "deleteText",
				SearchText: `\**hype\**`,
				Path:       "delete_text.txt",
			},
			wantMsg: "Deleted all occurrences of `\\**hype\\**` in `delete_text.txt`",
			out: `# HYPE ZONE
This file is .

## Hype Section
This section is pretty .
`,
		},
	}

	td := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(td, tt.cfg.Path)
			if err := os.WriteFile(path, []byte(tt.in), os.ModePerm); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			a, err := action.Parse(tt.cfg)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			msg, err := a.Run(pathTarget(td), action.Arguments{Variables: tt.vars})
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if msg != tt.wantMsg {
				t.Errorf("got message\n\t%s\nwant\n\t%s", msg, tt.wantMsg)
			}

			// Check that the file was modified correctly
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			got := string(data)
			if got != tt.out {
				t.Errorf("got file\n\t%s\nwant\n\t%s", got, tt.out)
			}
		})
	}
}

func TestTextActionError(t *testing.T) {
	tests := []struct {
		name string
		in   string
		cfg  action.Config
		vars map[string]string
	}{
		{
			name: "invalid regex",
			in:   inputText,
			cfg: action.Config{
				Type:       "replaceText",
				ApplyText:  "noop",
				SearchText: "($*^",
				Path:       "invalid_regex.md",
			},
		},
	}

	td := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(td, tt.cfg.Path)
			if err := os.WriteFile(path, []byte(tt.in), os.ModePerm); err != nil {
				t.Fatalf("failed to write file: %v", err)
			}
			a, err := action.Parse(tt.cfg)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			_, err = a.Run(pathTarget(td), action.Arguments{Variables: tt.vars})
			if err == nil {
				t.Error("want non-nil error", err)
			}
		})
	}
}

func TestFileAction(t *testing.T) {
	// Where source files will be created
	sd := t.TempDir()
	tests := []struct {
		name     string
		in       string
		existing string
		cfg      action.Config
		vars     map[string]string
		wantMsg  string
		out      string
	}{
		{
			name: "create file",
			in:   inputText,
			cfg: action.Config{
				Type:    "createFile",
				SrcPath: filepath.Join(sd, "create_file.md"),
				DstPath: "create_file.md",
			},
			wantMsg: "Created file `create_file.md`",
			out:     inputText,
		},
		{
			name:     "replace file",
			in:       inputText,
			existing: `content goes here`,
			cfg: action.Config{
				Type:    "replaceFile",
				SrcPath: filepath.Join(sd, "replace_file.md"),
				DstPath: "replace_file.md",
			},
			wantMsg: "Replaced file `replace_file.md`",
			out:     inputText,
		},
		{
			name: "createOrReplace missing file",
			in:   inputText,
			cfg: action.Config{
				Type:    "createOrReplaceFile",
				SrcPath: filepath.Join(sd, "replace_missing_file.md"),
				DstPath: "replace_missing_file.md",
			},
			wantMsg: "Created file `replace_missing_file.md`",
			out:     inputText,
		},
		{
			name:     "createOrReplace existing file",
			in:       inputText,
			existing: `content goes here`,
			cfg: action.Config{
				Type:    "createOrReplaceFile",
				SrcPath: filepath.Join(sd, "replace_existing_file.md"),
				DstPath: "replace_existing_file.md",
			},
			wantMsg: "Replaced file `replace_existing_file.md`",
			out:     inputText,
		},
		{
			name:     "delete file",
			existing: `content goes here`,
			cfg: action.Config{
				Type:    "deleteFile",
				DstPath: "delete_file.md",
			},
			wantMsg: "Deleted file `delete_file.md`",
		},
	}

	td := t.TempDir()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create source file
			if tt.in != "" {
				if err := os.WriteFile(tt.cfg.SrcPath, []byte(tt.in), os.ModePerm); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}
			// Create existing target file
			path := filepath.Join(td, tt.cfg.DstPath)
			if tt.existing != "" {
				if err := os.WriteFile(path, []byte(tt.existing), os.ModePerm); err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			a, err := action.Parse(tt.cfg)
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			msg, err := a.Run(pathTarget(td), action.Arguments{Variables: tt.vars})
			if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			if msg != tt.wantMsg {
				t.Errorf("got message\n\t%s\nwant\n\t%s", msg, tt.wantMsg)
			}

			// Check that file was deleted
			if tt.out == "" {
				if file.Exists(path) {
					t.Errorf("want file %s to not exists, but it does", path)
				}
				return
			}

			// Check that the file exists, with the correct contents
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}
			got := string(data)
			if got != tt.out {
				t.Errorf("got file\n\t%s\nwant\n\t%s", got, tt.out)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []struct {
		name string
		cfg  action.Config
	}{
		{
			name: "invalid text action",
			cfg: action.Config{
				Type:    "garlicText",
				SrcPath: "noop",
				DstPath: "noop",
				Path:    "noop.md",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Parse(tt.cfg)
			if err == nil {
				t.Error("want non-nil error", err)
			}
		})
	}
}
