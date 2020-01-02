package util

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDirExists(t *testing.T) {
	path := "../_fixtures/testdir"
	assert.True(t, FileOrDirExists(path))
}

func TestFileExists(t *testing.T) {
	path := "../_fixtures/testdir/test.txt"
	assert.True(t, FileOrDirExists(path))
}

func TestFileNotExists(t *testing.T) {
	path := "../_fixtures/notafile.txt"
	assert.False(t, FileOrDirExists(path))
}

type yamlConfig struct {
	Repos []struct {
		Name string `yaml:"name"`
	} `yaml:"repos"`
	Actions []struct {
		Type string `yaml:"type"`
		Path string `yaml:"path"`
	} `yaml:"actions"`
}

func TestReadYaml(t *testing.T) {
	path := "../_fixtures/cannon.test.yml"
	var config yamlConfig
	err := ReadYaml(path, &config)

	assert := assert.New(t)

	assert.NoError(err)
	assert.Len(config.Repos, 1)
	assert.Equal(config.Repos[0].Name, "TouchBistro/cannon")
	assert.Len(config.Actions, 1)
	assert.Equal(config.Actions[0].Type, "replaceLine")
	assert.Equal(config.Actions[0].Path, ".env.example")
}

func TestReadYamlError(t *testing.T) {
	path := "../_fixtures/invalid_yml"
	var config yamlConfig
	err := ReadYaml(path, &config)

	assert.Error(t, err)
}

func TestYamlNoFile(t *testing.T) {
	path := "../_fixtures/notafile.yml"
	var config yamlConfig
	err := ReadYaml(path, &config)

	assert.Error(t, err)
}

func TestExecOutput(t *testing.T) {
	assert := assert.New(t)
	output, err := ExecOutput("echo", "Hello World")

	assert.NoError(err)
	assert.Equal("Hello World\n", output)
}

func TestExecOutputError(t *testing.T) {
	assert := assert.New(t)
	output, err := ExecOutput("notacmd")

	assert.Error(err)
	assert.Empty(output)
}

func TestExec(t *testing.T) {
	err := Exec("echo", ".", "Hello World")

	assert.NoError(t, err)
}

func TestExecError(t *testing.T) {
	err := Exec("notacmd", ".", "Hello World")

	assert.Error(t, err)
}
