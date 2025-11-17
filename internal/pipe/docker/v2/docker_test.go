package docker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, Base{}.String())
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"docker buildx"}, Base{}.Dependencies(nil))
}

func TestSkip(t *testing.T) {
	t.Run("set", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Skip(skips.Docker))
		require.True(t, Base{}.Skip(ctx))
	})
	t.Run("no dockers", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{})
		require.True(t, Base{}.Skip(ctx))
	})
	t.Run("don't skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		})
		require.False(t, Base{}.Skip(ctx))
	})
	t.Run("snapshot don't skip snapshot", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Snapshot)
		require.False(t, Snapshot{}.Skip(ctx))
	})
	t.Run("snapshot skip non snapshot", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{{}},
		})
		require.True(t, Snapshot{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "dockerv2",
		DockersV2:   []config.DockerV2{{}},
	})
	require.NoError(t, Base{}.Default(ctx))
	d := ctx.Config.DockersV2[0]
	require.NotEmpty(t, d.ID)
	require.NotEmpty(t, d.Dockerfile)
	require.NotEmpty(t, d.Tags)
	require.NotEmpty(t, d.Platforms)
	require.Equal(t, "true", d.SBOM)
}

func TestMakeContext(t *testing.T) {
	t.Run("no dockerfile", func(t *testing.T) {
		_, err := makeContext(testctx.New(), config.DockerV2{}, nil)
		testlib.AssertSkipped(t, err)
	})
	t.Run("dockerfile tmpl error", func(t *testing.T) {
		_, err := makeContext(testctx.New(), config.DockerV2{
			Dockerfile: "{{.Nope}}",
		}, nil)
		testlib.RequireTemplateError(t, err)
	})
	t.Run("simple", func(t *testing.T) {
		dir, err := makeContext(testctx.New(), config.DockerV2{
			Dockerfile: "./testdata/Dockerfile",
			ExtraFiles: []string{"./testdata/foo.conf"},
		}, []*artifact.Artifact{
			{
				Name:   "mybin",
				Path:   "./testdata/mybin",
				Goos:   "linux",
				Goarch: "arm",
				Goarm:  "7",
			},
			{
				Name:   "mybin",
				Path:   "./testdata/mybin",
				Goos:   "linux",
				Goarch: "amd64",
			},
		})
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.RemoveAll(dir)
		})
		require.FileExists(t, filepath.Join(dir, "Dockerfile"))
		require.FileExists(t, filepath.Join(dir, "linux/amd64/mybin"))
		require.FileExists(t, filepath.Join(dir, "linux/arm/v7/mybin"))
		require.FileExists(t, filepath.Join(dir, "testdata/foo.conf"))
	})
}

func TestPublishExtraArgs(t *testing.T) {
	ctx := testctx.New()

	t.Run("sbom disabled", func(t *testing.T) {
		args, err := Publish{}.extraArgs(ctx, config.DockerV2{
			SBOM: "{{ .IsSnapshot }}",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"--push"}, args)
	})
	t.Run("sbom enabled", func(t *testing.T) {
		args, err := Publish{}.extraArgs(ctx, config.DockerV2{
			SBOM: "{{ not .IsSnapshot }}",
		})
		require.NoError(t, err)
		require.Equal(t, []string{"--push", "--attest=type=sbom"}, args)
	})
	t.Run("tmpl err", func(t *testing.T) {
		_, err := Publish{}.extraArgs(ctx, config.DockerV2{
			SBOM: "{{ not .IsSn",
		})
		testlib.RequireTemplateError(t, err)
	})
}

func TestMakeArgs(t *testing.T) {
	t.Run("tmpl error", func(t *testing.T) {
		for name, mod := range map[string]func(d *config.DockerV2){
			"images":      func(d *config.DockerV2) { d.Images = []string{"{{.Nope}}"} },
			"tags":        func(d *config.DockerV2) { d.Tags = []string{"{{.Nope}}"} },
			"labels":      func(d *config.DockerV2) { d.Labels = map[string]string{"foo": "{{.Nope}}"} },
			"annotations": func(d *config.DockerV2) { d.Annotations = map[string]string{"foo": "{{.Nope}}"} },
			"build args":  func(d *config.DockerV2) { d.BuildArgs = map[string]string{"{{.Nope}}": "bar"} },
			"flags":       func(d *config.DockerV2) { d.Flags = []string{"{{.Nope}}"} },
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
	t.Run("no images", func(t *testing.T) {
		_, _, err := makeArgs(testctx.New(), config.DockerV2{
			Dockerfile: "a",
		}, nil)
		testlib.AssertSkipped(t, err)
	})
	t.Run("no tags", func(t *testing.T) {
		_, _, err := makeArgs(testctx.New(), config.DockerV2{
			Dockerfile: "a",
			Images:     []string{"ghcr.io/foo/bar"},
		}, nil)
		require.Error(t, err)
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
			Annotations: map[string]string{
				"ignored":   "  ",
				"  ":        "also ignored",
				"foo":       "{{.ProjectName}}",
				"index:zaz": "zaz",
			},
			Platforms: []string{"linux/amd64", "linux/arm64"},
			BuildArgs: map[string]string{
				"FOO":     "{{.Env.FOO}}",
				"ignored": "  ",
				"  ":      "also ignored",
			},
			Flags: []string{"--ulimit=1000"},
		}, []string{"--push", "--attest=type=sbom"})
		require.NoError(t, err)
		require.Equal(
			t,
			[]string{
				"buildx", "build",
				"--platform", "linux/amd64,linux/arm64",
				"-t", "bar/bar:latest",
				"-t", "bar/bar:v",
				"-t", "ghcr.io/foo/bar:latest",
				"-t", "ghcr.io/foo/bar:v",
				"--push",
				"--attest=type=sbom",
				"--iidfile=id.txt",
				"--label", "date=2025-08-19T00:00:00Z",
				"--label", "name=dockerv2",
				"--annotation", "index:foo=dockerv2",
				"--annotation", "index:zaz=zaz",
				"--build-arg", "FOO=bar",
				"--ulimit=1000",
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

func TestDisable(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{
				{
					Disable: "true",
				},
			},
		})
		require.NoError(t, Base{}.Default(ctx))
		testlib.AssertSkipped(t, Snapshot{}.Run(ctx))
		testlib.AssertSkipped(t, Publish{}.Publish(ctx))
	})
	t.Run("template error", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			DockersV2: []config.DockerV2{
				{
					Disable: "{{ .no }}",
				},
			},
		})
		require.NoError(t, Base{}.Default(ctx))
		testlib.RequireTemplateError(t, Snapshot{}.Run(ctx))
		testlib.RequireTemplateError(t, Publish{}.Publish(ctx))
	})
}

func TestToPlatform(t *testing.T) {
	for expected, art := range map[string]artifact.Artifact{
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
		"linux/arm/v5": {
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "5",
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

	t.Run("unsupported os", func(t *testing.T) {
		_, err := toPlatform(&artifact.Artifact{
			Goos: "nope",
		})
		require.Error(t, err)
	})

	t.Run("unsupported arch", func(t *testing.T) {
		_, err := toPlatform(&artifact.Artifact{
			Goos:   "linux",
			Goarch: "nope",
		})
		require.Error(t, err)
	})

	t.Run("unsupported arm", func(t *testing.T) {
		_, err := toPlatform(&artifact.Artifact{
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "4",
		})
		require.Error(t, err)
	})
}

func TestParsePlatform(t *testing.T) {
	for input, output := range map[string]platform{
		"linux/amd64":  {os: "linux", arch: "amd64"},
		"linux/arm/v6": {os: "linux", arch: "arm", arm: "6"},
	} {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, output, parsePlatform(input))
		})
	}
}

func TestContextArtifacts(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "dockerv2",
	})

	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Goos:   "linux",
		Goarch: "arm",
		Goarm:  "7",
		Type:   artifact.Binary,
		Extra: artifact.Extras{
			artifact.ExtraID: "id1",
		},
	})
	for _, arch := range []string{"amd64", "arm64"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Goos:   "linux",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: artifact.Extras{
				artifact.ExtraID: "id1",
			},
		})
	}

	arts := contextArtifacts(ctx, config.DockerV2{
		Platforms: []string{"linux/arm/v7", "linux/amd64", "linux/arm64"},
		IDs:       []string{"id1"},
	})
	require.Len(t, arts, 3)
}

func TestIsRetriableManifestCreate(t *testing.T) {
	require.True(t, isRetriableManifestCreate(gerrors.Wrap(nil, "", "output", "manifest verification failed for digest")))
	require.False(t, isRetriableManifestCreate(gerrors.Wrap(nil, "", "output", "some other error")))
	require.False(t, isRetriableManifestCreate(errors.New("some other error")))
	require.False(t, isRetriableManifestCreate(nil))
}

func TestTagSuffix(t *testing.T) {
	for plat, suffix := range map[string]string{
		"linux/amd64":   "amd64",
		"linux/arm64":   "arm64",
		"linux/arm/v7":  "armv7",
		"windows/amd64": "amd64",
	} {
		t.Run(plat, func(t *testing.T) {
			require.Equal(t, suffix, tagSuffix(plat))
		})
	}
}

func TestGetBuildxDriver(t *testing.T) {
	testlib.CheckDocker(t)
	
	ctx := testctx.New()
	driver := getBuildxDriver(ctx)
	
	// The driver should be one of the known types, or empty if buildx is not available
	// We just verify it doesn't crash and returns a string
	t.Logf("detected buildx driver: %q", driver)
	require.NotPanics(t, func() {
		getBuildxDriver(ctx)
	})
}
