package ko

import (
	"fmt"
	"testing"

	_ "github.com/distribution/distribution/v3/registry/auth/htpasswd"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

const (
	registryPort = "5052"
	registry     = "localhost:5052/"
)

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{
		Env: []string{
			"KO_DOCKER_REPO=" + registry,
			"COSIGN_REPOSITORY=" + registry,
			"LDFLAGS=foobar",
			"FLAGS=barfoo",
			"LE_ENV=test",
		},
		Builds: []config.Build{
			{
				ID: "default",
				BuildDetails: config.BuildDetails{
					Ldflags: []string{"{{.Env.LDFLAGS}}"},
					Flags:   []string{"{{.Env.FLAGS}}"},
					Env:     []string{"SOME_ENV={{.Env.LE_ENV}}"},
				},
			},
		},
		Kos: []config.Ko{
			{},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, config.Ko{
		ID:               "default",
		Build:            "default",
		BaseImage:        chainguardStatic,
		Repository:       registry,
		CosignRepository: registry,
		Platforms:        []string{"linux/amd64"},
		SBOM:             "spdx",
		Tags:             []string{"latest"},
		WorkingDir:       ".",
		Ldflags:          []string{"{{.Env.LDFLAGS}}"},
		Flags:            []string{"{{.Env.FLAGS}}"},
		Env:              []string{"SOME_ENV={{.Env.LE_ENV}}"},
	}, ctx.Config.Kos[0])
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPublishPipe(t *testing.T) {
	testlib.StartRegistry(t, "ko_registry", registryPort)

	ctx := &context.Context{
		Parallelism: 1,
		Config: config.Project{
			Builds: []config.Build{
				{
					ID: "foo",
					BuildDetails: config.BuildDetails{
						Ldflags: []string{"-s", "-w"},
						Flags:   []string{"-tags", "netgo"},
						Env:     []string{"GOCACHE=" + t.TempDir()},
					},
				},
			},
			Kos: []config.Ko{
				{
					ID:         "default",
					Build:      "foo",
					WorkingDir: "./testdata/app/",
					BaseImage:  "cgr.dev/chainguard/static",
					Repository: fmt.Sprintf("%s/goreleasertest", registry),
					Platforms:  []string{"linux/amd64"},
					Tags:       []string{"latest"},
				},
			},
		},
	}

	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Publish(ctx))
}
