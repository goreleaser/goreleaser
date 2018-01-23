package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

var emptyEnv []string

type dummy struct{}

func (*dummy) Default(build config.Build) config.Build {
	return build
}
func (*dummy) Build(ctx *context.Context, build config.Build, options Options) error {
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	var builder = &dummy{}
	Register("dummy", builder)
	assert.Equal(t, builder, For("dummy"))
}

func TestRun(t *testing.T) {
	assert.NoError(t, Run(
		context.New(config.Project{}),
		[]string{"go", "list", "./..."},
		emptyEnv,
	))
}

func TestRunInvalidCommand(t *testing.T) {
	assert.Error(t, Run(
		context.New(config.Project{}),
		[]string{"gggggo", "nope"},
		emptyEnv,
	))
}
