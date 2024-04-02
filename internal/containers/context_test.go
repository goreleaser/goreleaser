package containers

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func Test_processImageTemplates(t *testing.T) {
	ctx := testctx.NewWithCfg(
		config.Project{
			Builds: []config.Build{
				{
					ID: "default",
				},
			},
			Dockers: []config.Docker{
				{
					ImageDefinition: config.ImageDefinition{
						Dockerfile: "Dockerfile.foo",
						ImageTemplates: []string{
							"user/image:{{.Tag}}",
							"gcr.io/image:{{.Tag}}-{{.Env.FOO}}",
							"gcr.io/image:v{{.Major}}.{{.Minor}}",
						},
					},
					SkipPush: "true",
				},
			},
			Env: []string{"FOO=123"},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithCommit("a1b2c3d4"),
		testctx.WithSemver(1, 0, 0, ""),
	)
	require.Len(t, ctx.Config.Dockers, 1)

	docker := ctx.Config.Dockers[0]
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)

	images, err := processImageTemplates(ctx, docker.ImageTemplates)
	require.NoError(t, err)
	require.Equal(t, []string{
		"user/image:v1.0.0",
		"gcr.io/image:v1.0.0-123",
		"gcr.io/image:v1.0",
	}, images)
}
