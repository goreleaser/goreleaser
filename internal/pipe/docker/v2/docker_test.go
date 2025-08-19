package docker

import (
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"docker"}, Pipe{}.Dependencies(nil))
}

func TestSkip(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Skip(skips.Docker))
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("no dockers", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{})
		require.True(t, Pipe{}.Skip(ctx))
	})
	t.Run("don't skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "dockerv2",
		DockersV2:   []config.DockerV2{{}},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	d := ctx.Config.DockersV2[0]
	require.NotEmpty(t, d.ID)
	require.NotEmpty(t, d.Dockerfile)
	require.NotEmpty(t, d.Tags)
	require.NotEmpty(t, d.Platforms)
}

func TestMakeArgs(t *testing.T) {
	t.Run("tmpl error", func(t *testing.T) {
		for name, mod := range map[string]func(d *config.DockerV2){
			"dockerfile": func(d *config.DockerV2) { d.Dockerfile = "{{.Nope}}" },
			"images":     func(d *config.DockerV2) { d.Images = []string{"{{.Nope}}"} },
			"tags":       func(d *config.DockerV2) { d.Tags = []string{"{{.Nope}}"} },
			"labels":     func(d *config.DockerV2) { d.Labels = map[string]string{"foo": "{{.Nope}}"} },
			"build args": func(d *config.DockerV2) { d.BuildArgs = map[string]string{"{{.Nope}}": "bar"} },
		} {
			t.Run(name, func(t *testing.T) {
				ctx := testctx.New()
				d := config.DockerV2{
					Dockerfile: "Dockerfile",
					Images:     []string{"ghcr.io/foo/bar"},
					Tags:       []string{"latest", "v{{.Version}}"},
				}
				mod(&d)
				_, _, err := makeArgs(ctx, d, nil)
				testlib.RequireTemplateError(t, err)
			})
		}
	})
	t.Run("no dockerfile", func(t *testing.T) {
		_, _, err := makeArgs(testctx.New(), config.DockerV2{}, nil)
		testlib.AssertSkipped(t, err)
	})
	t.Run("no images", func(t *testing.T) {
		_, _, err := makeArgs(testctx.New(), config.DockerV2{
			Dockerfile: "a",
		}, nil)
		testlib.AssertSkipped(t, err)
	})
	t.Run("no tags", func(t *testing.T) {
		_, images, err := makeArgs(testctx.New(), config.DockerV2{
			Dockerfile: "a",
			Images:     []string{"ghcr.io/foo/bar"},
		}, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"ghcr.io/foo/bar:latest"}, images)
	})
	t.Run("simple", func(t *testing.T) {
		ctx := testctx.NewWithCfg(
			config.Project{
				ProjectName: "dockerv2",
			},
			testctx.WithEnv(map[string]string{"FOO": "bar"}),
			testctx.WithDate(time.Date(2025, 8, 19, 0, 0, 0, 0, time.UTC)),
		)
		args, images, err := makeArgs(ctx, config.DockerV2{
			ID:         "test",
			IDs:        []string{"test"},
			Dockerfile: "{{.Env.FOO}}.dockerfile",
			Images:     []string{"{{.Env.FOO}}/bar", "ghcr.io/foo/bar"},
			Tags:       []string{"latest", "v{{.Version}}", "{{ if .IsNightly }}nightly{{ end }}"},
			Labels: map[string]string{
				"date":    "{{.Date}}",
				"ignored": "  ",
				"  ":      "also ignored",
				"name":    "{{.ProjectName}}",
			},
			Platforms: []string{"linux/amd64", "linux/arm64"},
			BuildArgs: map[string]string{
				"FOO":     "{{.Env.FOO}}",
				"ignored": "  ",
				"  ":      "also ignored",
			},
		}, []string{"--push"})
		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"buildx", "build",
				"--platform", "linux/amd64,linux/arm64",
				"--attest=type=sbom",
				"-t", "bar/bar:latest",
				"-t", "bar/bar:v",
				"-t", "ghcr.io/foo/bar:latest",
				"-t", "ghcr.io/foo/bar:v",
				"--push",
				"--label", "date=2025-08-19T00:00:00Z",
				"--label", "name=dockerv2",
				"--build-arg", "FOO=bar",
				"-f", "bar.dockerfile",
				".",
			},
			args,
		)
		require.Equal(
			t,
			[]string{
				"bar/bar:latest",
				"bar/bar:v",
				"ghcr.io/foo/bar:latest",
				"ghcr.io/foo/bar:v",
			},
			images,
		)
	})
}

func TestPlatform(t *testing.T) {
	for expected, art := range map[string]artifact.Artifact{
		"darwin/amd64": {
			Goos:   "darwin",
			Goarch: "amd64",
		},
		"darwin/arm64": {
			Goos:   "darwin",
			Goarch: "arm64",
		},
		"windows/amd64": {
			Goos:   "windows",
			Goarch: "amd64",
		},
		"windows/arm64": {
			Goos:   "windows",
			Goarch: "arm64",
		},
		"linux/amd64": {
			Goos:   "linux",
			Goarch: "amd64",
		},
		"linux/arm64": {
			Goos:   "linux",
			Goarch: "arm64",
		},
		"linux/arm/v7": {
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "7",
		},
		"linux/arm/v6": {
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
		},
		"linux/386": {
			Goos:   "linux",
			Goarch: "386",
		},
		"linux/ppc64le": {
			Goos:   "linux",
			Goarch: "ppc64le",
		},
		"linux/s390x": {
			Goos:   "linux",
			Goarch: "s390x",
		},
		"linux/riscv64": {
			Goos:   "linux",
			Goarch: "riscv64",
		},
	} {
		t.Run(expected, func(t *testing.T) {
			plat, err := toPlatform(&art)
			require.NoError(t, err)
			require.Equal(t, expected, plat)
		})
	}
}
