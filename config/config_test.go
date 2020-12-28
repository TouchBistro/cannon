package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/TouchBistro/cannon/action"
	"github.com/TouchBistro/cannon/config"
)

func TestRepoBranch(t *testing.T) {
	tests := []struct {
		name string
		repo config.Repo
		want string
	}{
		{
			"default base branch",
			config.Repo{
				Name: "TouchBistro/cannon",
			},
			"master",
		},
		{
			"custom base branch",
			config.Repo{
				Name: "TouchBistro/example",
				Base: "develop",
			},
			"develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.repo.BaseBranch()
			if got != tt.want {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestInitConfig(t *testing.T) {
	tmpdir := t.TempDir()
	os.Setenv("HOME", tmpdir)

	reader := strings.NewReader(`repos:
  - name: TouchBistro/cannon
  - name: TouchBistro/example
    base: develop
actions:
  - type: replaceLine
    source: DB_USER=core
    target: DB_USER=SA
    path: .env.example
  - type: runCommand
    run: yarn
`)

	err := config.Init(reader)
	if err != nil {
		t.Errorf("want nil error, got %v", err)
	}

	wantDir := filepath.Join(tmpdir, ".cannon")
	gotDir := config.CannonDir()
	if gotDir != wantDir {
		t.Errorf("got %s, want %s", gotDir, wantDir)
	}
	stat, err := os.Stat(gotDir)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", gotDir, err)
	}
	if !stat.IsDir() {
		t.Errorf("want %s to be a dir, got %s", gotDir, stat.Mode())
	}

	wantConfig := config.CannonConfig{
		Repos: []config.Repo{
			{
				Name: "TouchBistro/cannon",
			},
			{
				Name: "TouchBistro/example",
				Base: "develop",
			},
		},
		Actions: []action.Action{
			{
				Type:   action.ActionReplaceLine,
				Source: "DB_USER=core",
				Target: "DB_USER=SA",
				Path:   ".env.example",
			},
			{
				Type: action.ActionRunCommand,
				Run:  "yarn",
			},
		},
	}
	gotConfig := *config.Config()
	if !reflect.DeepEqual(gotConfig, wantConfig) {
		t.Errorf("got %+v, want %+v", gotConfig, wantConfig)
	}
}
