package version

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestVersionApplication(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Version: "1.0.0",
		},
	}

	versionPipe := Pipe{}
	versionPipe.Run(ctx)

	require.Equal(t, ctx.Version, ctx.Config.Version)
}

func TestEnvironmentApply(t *testing.T) {
	ctx := &context.Context{
		Env: map[string]string{
			"VersionOverride": "1.0.0",
		},
		Config: config.Project{
			Version: "{{.Env.VersionOverride}}",
		},
	}

	versionPipe := Pipe{}
	versionPipe.Run(ctx)

	require.Equal(t, ctx.Version, "1.0.0")
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
