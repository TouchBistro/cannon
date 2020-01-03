package action

import (
	"testing"

	"github.com/TouchBistro/cannon/config"
	"github.com/stretchr/testify/assert"
)

func TestExpandRepoVar(t *testing.T) {
	assert := assert.New(t)
	expanded := expandRepoVar("container_name: $REPONAME_container", "node-boilerplate")
	notExpanded := expandRepoVar("NODE_ENV=development", "node-boilerplate")

	assert.Equal("container_name: node-boilerplate_container", expanded)
	assert.Equal("NODE_ENV=development", notExpanded)
}

func TestReplaceLine(t *testing.T) {
	assert := assert.New(t)
	action := config.Action{
		Source: "NODE_ENV=$REPONAME",
		Target: "NODE_ENV=development",
	}
	input := []byte("NODE_ENV=development\n")
	outputData, msg, err := ReplaceLine(action, "node-boilerplate", input)

	assert.Equal(outputData, []byte("NODE_ENV=node-boilerplate\n"))
	assert.NotEmpty(msg)
	assert.NoError(err)
}

func TestReplaceLineRegex(t *testing.T) {
	assert := assert.New(t)
	action := config.Action{
		Source: "NODE_ENV=test",
		Target: "NODE_ENV=([a-zA-Z-]+)",
	}
	input := []byte("NODE_ENV=random\n")
	outputData, msg, err := ReplaceLine(action, "node-boilerplate", input)

	assert.Equal("NODE_ENV=test\n", string(outputData))
	assert.NotEmpty(msg)
	assert.NoError(err)
}

func TestRepaceLineError(t *testing.T) {
	assert := assert.New(t)
	action := config.Action{
		Source: "NODE_ENV=test",
		Target: "NODE_ENV=($&^",
	}
	input := []byte("NODE_ENV=development\n")
	outputData, msg, err := ReplaceLine(action, "node-boilerplate", input)

	assert.Nil(outputData)
	assert.Empty(msg)
	assert.Error(err)
}

func TestDeleteLine(t *testing.T) {
	assert := assert.New(t)
	action := config.Action{
		Target: "NODE_ENV=test",
	}
	input := []byte("PORT=8080\nNODE_ENV=test\n")
	outputData, msg, err := DeleteLine(action, "node-boilerplate", input)

	assert.Equal("PORT=8080\n", string(outputData))
	assert.NotEmpty(msg)
	assert.NoError(err)
}
