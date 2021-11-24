package shell_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/goreleaser/goreleaser/internal/shell"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

func TestRunAValidCommand(t *testing.T) {
	assert := assert.New(t)

	ctx := context.New(config.Project{})

	err := shell.Run(ctx, "", []string{"echo", "test"}, []string{})
	assert.NoError(err)
}

func TestRunAnInValidCommand(t *testing.T) {
	assert := assert.New(t)

	ctx := context.New(config.Project{})

	err := shell.Run(ctx, "", []string{"invalid", "command"}, []string{})

	assert.Error(err)
	assert.Contains(err.Error(), "executable file not found")
}
