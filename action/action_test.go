package action

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const textTests = "testdata/text_tests"
const inputText = `# HYPE ZONE
This file is ***hype***.

## Hype Section
This section is pretty hype.
`

type MockWriter struct {
	w *bytes.Buffer
}

func mockFile(contents string) (*strings.Reader, *MockWriter) {
	r := strings.NewReader(contents)
	mw := &MockWriter{w: bytes.NewBufferString(contents)}
	return r, mw
}

func (mw *MockWriter) Truncate(size int64) error {
	mw.w.Truncate(int(size))
	return nil
}

func (mw *MockWriter) WriteAt(b []byte, off int64) (n int, err error) {
	return mw.w.Write(b)
}

func (mw *MockWriter) String() string {
	return mw.w.String()
}

func TestReplaceLine(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionReplaceLine,
		Source: "# WOKE ZONE",
		Target: "# HYPE ZONE",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.NotEmpty(msg)
	assert.NoError(err)

	assert.Equal(`# WOKE ZONE
This file is ***hype***.

## Hype Section
This section is pretty hype.
`, mw.String())
}

func TestDeleteLine(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionDeleteLine,
		Target: "## Hype Section",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.NotEmpty(msg)
	assert.NoError(err)

	assert.Equal(`# HYPE ZONE
This file is ***hype***.

This section is pretty hype.
`, mw.String())
}

func TestReplaceText(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionReplaceText,
		Source: "*****",
		Target: "^#.+",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.NotEmpty(msg)
	assert.NoError(err)

	assert.Equal(`*****
This file is ***hype***.

*****
This section is pretty hype.
`, mw.String())
}

func TestAppendText(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionAppendText,
		Source: " --- ${REPO_OWNER} - ${REPO_NAME}",
		Target: "^#.+",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.NotEmpty(msg)
	assert.NoError(err)

	assert.Equal(`# HYPE ZONE --- TouchBistro - node-boilerplate
This file is ***hype***.

## Hype Section --- TouchBistro - node-boilerplate
This section is pretty hype.
`, mw.String())
}

func TestDeleteText(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionDeleteText,
		Target: `\**hype\**`,
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.NotEmpty(msg)
	assert.NoError(err)

	assert.Equal(`# HYPE ZONE
This file is .

## Hype Section
This section is pretty .
`, mw.String())
}

func TestInvalidRegex(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   ActionReplaceText,
		Source: "noop",
		Target: "($*^",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.Empty(msg)
	assert.Error(err)
}

func TestInvalidTextAction(t *testing.T) {
	assert := assert.New(t)
	action := Action{
		Type:   "ActionGarlicText",
		Source: "noop",
		Target: "noop",
		Path:   textTests + "/hype.md",
	}

	r, mw := mockFile(inputText)
	msg, err := ExecuteTextAction(action, r, mw, "TouchBistro/node-boilerplate")

	assert.Empty(msg)
	assert.Error(err)
}
