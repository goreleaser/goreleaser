package gentoo

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunReleaseDisabled(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			Disable: "true",
		},
		Gentoos: []config.Gentoo{
			{
				Repository: config.RepoRef{
					Owner: "test",
					Name:  "test",
				},
			},
		},
	}, testctx.WithVersion("1.0.0"))
	ctx.TokenType = context.TokenTypeGitHub
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "foo_1.0.0_linux_amd64.tar.gz",
		Path:    "amd64.tar.gz",
		Goos:    "linux",
		Goarch:  "amd64",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
	})

	err := Pipe{}.Run(ctx)
	require.ErrorIs(t, err, client.ErrReleaseDisabled)
}
