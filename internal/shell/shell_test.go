package shell_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/goreleaser/goreleaser/internal/shell"
)

func TestRunAValidCommand(t *testing.T) {
	assert := assert.New(t)

	err := shell.Run("", []string{"echo", "test"}, []string{})
	assert.NoError(err)
}

func TestRunAnInValidCommand(t *testing.T) {
	assert := assert.New(t)

	err := shell.Run("", []string{"invalid", "command"}, []string{})

	assert.Error(err)
	assert.Contains(err.Error(), "executable file not found")
}

func TestRunAValidCommandWithOutput(t *testing.T) {
	assert := assert.New(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := shell.RunWithOutput("", []string{"echo", "test"}, []string{}, stdout, stderr)
	assert.NoError(err)
	assert.Equal("test\n", stdout.String())
	assert.Empty(stderr.String())
}
