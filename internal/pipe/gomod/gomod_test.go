package gomod

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser", ctx.ModulePath)
}

func TestRunOutsideGoModule(t *testing.T) {
	require.NoError(t, os.Chdir(t.TempDir()))
	ctx := context.New(config.Project{})
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ModulePath)
}

func TestRunCommandError(t *testing.T) {
	os.Unsetenv("PATH")
	require.NoError(t, os.Chdir(t.TempDir()))
	ctx := context.New(config.Project{})
	require.EqualError(t, Pipe{}.Run(ctx), "failed to get module path: exec: \"go\": executable file not found in $PATH: ")
	require.Empty(t, ctx.ModulePath)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
