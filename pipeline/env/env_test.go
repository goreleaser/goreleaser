package env

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestValidEnv(t *testing.T) {
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestInvalidEnv(t *testing.T) {
	assert := assert.New(t)
	assert.NoError(os.Unsetenv("GITHUB_TOKEN"))
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}
