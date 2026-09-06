package docker

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Skip(skips.Docker))
		require.True(t, Base{}.Skip(ctx))
	})
	t.Run("no dockers", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
		require.True(t, Base{}.Skip(ctx))
	})
	t.Run("don't skip", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		})
		require.False(t, Base{}.Skip(ctx))
	})
	t.Run("snapshot don't skip snapshot", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Snapshot)
		require.False(t, Snapshot{}.Skip(ctx))
	})
	t.Run("snapshot skip non snapshot", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		})
		require.True(t, Snapshot{}.Skip(ctx))
	})
	t.Run("snapshot skip non snapshot with publish skipped", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Skip(skips.Publish))
		require.True(t, Snapshot{}.Skip(ctx))
	})
	t.Run("snapshot skip docker skipped", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockersV2: []config.DockerV2{{}},
		}, testctx.Skip(skips.Docker), testctx.Snapshot)
		require.True(t, Snapshot{}.Skip(ctx))
	})
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		_, err := makeContext(config.DockerV2{}, nil, "  ")
		testlib.AssertSkipped(t, err)
	})
	t.Run("simple", func(t *testing.T) {
		dir, err := makeContext(config.DockerV2{
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
		}, "./testdata/Dockerfile")
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

func TestMakeContextRemovesTemporaryDirOnCopyError(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, dir string) (config.DockerV2, []*artifact.Artifact){
		"extra file": func(t *testing.T, dir string) (config.DockerV2, []*artifact.Artifact) {
			t.Helper()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "assets.bin"), []byte(strings.Repeat("a", 8192)), 0o644))
			return config.DockerV2{
				ID:         "test",
				ExtraFiles: []string{"assets.bin", "missing.txt"},
			}, nil
		},
		"artifact": func(t *testing.T, dir string) (config.DockerV2, []*artifact.Artifact) {
			t.Helper()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "mybin"), []byte("binary"), 0o644))
			return config.DockerV2{ID: "test"}, []*artifact.Artifact{
				{
					Name:   "mybin",
					Path:   "mybin",
					Goos:   "linux",
					Goarch: "amd64",
				},
				{
					Name:   "missing",
					Path:   "missing",
					Goos:   "linux",
					Goarch: "amd64",
				},
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			dir, tmp := isolatedDockerContextTemp(t)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
			d, artifacts := setup(t, dir)

			wd, err := makeContext(d, artifacts, "Dockerfile")

			require.Error(t, err)
			require.Empty(t, wd)
			require.Empty(t, dockerContextDirs(t, tmp))
		})
	}
}

func TestBuildImageRemovesTemporaryDirOnSuccess(t *testing.T) {
	dir, tmp := isolatedDockerContextTemp(t)
	fakeDockerBuildxBuild(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mybin"), []byte("binary"), 0o644))

	ctx := testctx.Wrap(t.Context())
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "mybin",
		Goos:   "linux",
		Goarch: "amd64",
		Type:   artifact.Binary,
	})

	require.NoError(t, buildImage(ctx, config.DockerV2{
		ID:         "test",
		Dockerfile: "Dockerfile",
		Images:     []string{"ghcr.io/foo/bar"},
		Tags:       []string{"latest"},
		Platforms:  []string{"linux/amd64"},
	}, "--load"))

	require.Empty(t, dockerContextDirs(t, tmp))
}

func TestMakeContextUsesStableAmd64Variant(t *testing.T) {
	for name, order := range map[string][]string{
		"v1 then v3": {"v1", "v3"},
		"v3 then v1": {"v3", "v1"},
	} {
		t.Run(name, func(t *testing.T) {
			dir, _ := isolatedDockerContextTemp(t)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644))
			for _, goamd64 := range []string{"v1", "v3"} {
				require.NoError(t, os.WriteFile(filepath.Join(dir, "mybin-"+goamd64), []byte(goamd64), 0o644))
			}

			ctx := testctx.Wrap(t.Context())
			for _, goamd64 := range order {
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    "mybin",
					Path:    "mybin-" + goamd64,
					Goos:    "linux",
					Goarch:  "amd64",
					Goamd64: goamd64,
					Type:    artifact.Binary,
					Extra: artifact.Extras{
						artifact.ExtraID: "cli",
					},
				})
			}

			d := config.DockerV2{
				ID:        "test",
				IDs:       []string{"cli"},
				Platforms: []string{"linux/amd64"},
			}
			wd, err := makeContext(d, contextArtifacts(ctx, d), "Dockerfile")
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = os.RemoveAll(wd)
			})

			content, err := os.ReadFile(filepath.Join(wd, "linux/amd64/mybin"))
			require.NoError(t, err)
			require.Equal(t, "v1", string(content))
		})
	}
}

func isolatedDockerContextTemp(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "work")
	tmp := filepath.Join(root, "tmp")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.MkdirAll(tmp, 0o755))
	t.Setenv("TMPDIR", tmp)
	t.Chdir(dir)
	return dir, tmp
}

func dockerContextDirs(t *testing.T, tmp string) []string {
	t.Helper()
	entries, err := os.ReadDir(tmp)
	require.NoError(t, err)
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "goreleaserdocker") {
			dirs = append(dirs, entry.Name())
		}
	}
	return dirs
}

func fakeDockerBuildxBuild(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	name := "docker"
	script := "#!/bin/sh\nprintf 'sha256:test' > id.txt\n"
	if testlib.IsWindows() {
		name = "docker.bat"
		script = "@echo off\r\necho sha256:test> id.txt\r\n"
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestPublishExtraArgs(t *testing.T) {
	ctx := testctx.Wrap(t.Context())

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
			"dockerfile":  func(d *config.DockerV2) { d.Dockerfile = "{{.Nope}}" },
			"images":      func(d *config.DockerV2) { d.Images = []string{"{{.Nope}}"} },
			"tags":        func(d *config.DockerV2) { d.Tags = []string{"{{.Nope}}"} },
			"labels":      func(d *config.DockerV2) { d.Labels = map[string]string{"foo": "{{.Nope}}"} },
			"annotations": func(d *config.DockerV2) { d.Annotations = map[string]string{"foo": "{{.Nope}}"} },
			"build args":  func(d *config.DockerV2) { d.BuildArgs = map[string]string{"{{.Nope}}": "bar"} },
			"flags":       func(d *config.DockerV2) { d.Flags = []string{"{{.Nope}}"} },
		} {
			t.Run(name, func(t *testing.T) {
				ctx := testctx.Wrap(t.Context())
				d := config.DockerV2{
					Dockerfile: "Dockerfile",
					Images:     []string{"ghcr.io/foo/bar"},
					Tags:       []string{"latest", "v{{.Version}}"},
				}
				mod(&d)
				_, err := makeArgs(ctx, d, nil)
				testlib.RequireTemplateError(t, err)
			})
		}
	})
	t.Run("no images", func(t *testing.T) {
		_, err := makeArgs(testctx.Wrap(t.Context()), config.DockerV2{
			Dockerfile: "a",
		}, nil)
		testlib.AssertSkipped(t, err)
	})
	t.Run("no tags", func(t *testing.T) {
		_, err := makeArgs(testctx.Wrap(t.Context()), config.DockerV2{
			Dockerfile: "a",
			Images:     []string{"ghcr.io/foo/bar"},
		}, nil)
		require.Error(t, err)
	})
	t.Run("no dockerfile", func(t *testing.T) {
		da, err := makeArgs(testctx.Wrap(t.Context()), config.DockerV2{
			Images: []string{"ghcr.io/foo/bar"},
			Tags:   []string{"latest"},
		}, nil)
		require.NoError(t, err)
		require.Empty(t, da.dockerfile)
	})
	t.Run("simple", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(
			t.Context(),
			config.Project{
				ProjectName: "dockerv2",
			},
			testctx.WithEnv(map[string]string{"FOO": "bar"}),
			testctx.WithDate(time.Date(2025, 8, 19, 0, 0, 0, 0, time.UTC)),
		)
		da, err := makeArgs(ctx, config.DockerV2{
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
			da.args,
		)
		require.Equal(
			t,
			[]string{
				"bar/bar:latest",
				"bar/bar:v",
				"ghcr.io/foo/bar:latest",
				"ghcr.io/foo/bar:v",
			},
			da.images,
		)
	})
}

func TestMakeArgsAnnotationScopes(t *testing.T) {
	multi := []string{"linux/amd64", "linux/arm64"}
	for name, tt := range map[string]struct {
		annotations map[string]string
		platforms   []string
		expect      []string
	}{
		"unscoped defaults to index": {
			annotations: map[string]string{"foo": "bar"},
			platforms:   multi,
			expect:      []string{"index:foo=bar"},
		},
		"explicit index is not doubled": {
			annotations: map[string]string{"index:foo": "bar"},
			platforms:   multi,
			expect:      []string{"index:foo=bar"},
		},
		"manifest scope is kept": {
			annotations: map[string]string{"manifest:foo": "bar"},
			platforms:   multi,
			expect:      []string{"manifest:foo=bar"},
		},
		"multiple scopes are kept": {
			annotations: map[string]string{"index,manifest:foo": "bar"},
			platforms:   multi,
			expect:      []string{"index,manifest:foo=bar"},
		},
		"platform qualified scope is kept": {
			annotations: map[string]string{"manifest[linux/amd64]:foo": "bar"},
			platforms:   multi,
			expect:      []string{"manifest[linux/amd64]:foo=bar"},
		},
		"descriptor scopes are kept": {
			annotations: map[string]string{
				"index-descriptor:foo":    "bar",
				"manifest-descriptor:baz": "qux",
			},
			platforms: multi,
			expect: []string{
				"index-descriptor:foo=bar",
				"manifest-descriptor:baz=qux",
			},
		},
		"colon in value is not a scope": {
			annotations: map[string]string{"foo": "https://example.com"},
			platforms:   multi,
			expect:      []string{"index:foo=https://example.com"},
		},
		"unknown scope is part of the key": {
			annotations: map[string]string{"bogus:foo": "bar"},
			platforms:   multi,
			expect:      []string{"index:bogus:foo=bar"},
		},
		"one unknown scope in the list invalidates it": {
			annotations: map[string]string{"index,bogus:foo": "bar"},
			platforms:   multi,
			expect:      []string{"index:index,bogus:foo=bar"},
		},
		"single platform is left alone": {
			annotations: map[string]string{"foo": "bar"},
			platforms:   []string{"linux/amd64"},
			expect:      []string{"foo=bar"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			da, err := makeArgs(testctx.Wrap(t.Context()), config.DockerV2{
				Dockerfile:  "Dockerfile",
				Images:      []string{"ghcr.io/foo/bar"},
				Tags:        []string{"latest"},
				Platforms:   tt.platforms,
				Annotations: tt.annotations,
			}, nil)
			require.NoError(t, err)
			require.Equal(t, tt.expect, annotationsOf(da.args))
		})
	}
}

func annotationsOf(args []string) []string {
	var result []string
	for i, arg := range args {
		if arg == "--annotation" {
			result = append(result, args[i+1])
		}
	}
	return result
}

func TestDisable(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func TestIsDockerDaemonAvailableNoDaemon(t *testing.T) {
	// loopback refuses the connection at once. A missing unix socket makes the
	// docker CLI on windows think about it for 5.46s.
	t.Setenv("DOCKER_HOST", "tcp://127.0.0.1:1")
	require.False(t, isDockerDaemonAvailable(t.Context()))
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
		"linux/amd64/v3": {
			Goos:    "linux",
			Goarch:  "amd64",
			Goamd64: "v3",
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
		"linux/amd64":    {os: "linux", arch: "amd64", amd64: "v1"},
		"linux/amd64/v3": {os: "linux", arch: "amd64", amd64: "v3"},
		"linux/arm/v6":   {os: "linux", arch: "arm", arm: "6"},
		"linux/arm64/v8": {os: "linux", arch: "arm64", arm64: "v8.0"},
		"linux":          {os: "linux"},
	} {
		t.Run(input, func(t *testing.T) {
			require.Equal(t, output, parsePlatform(input))
		})
	}
}

func TestContextArtifacts(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

	for _, id := range []string{"id1", "id2"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   id + "-1.0.0-py3-none-any.whl",
			Goos:   "all",
			Goarch: "all",
			Type:   artifact.PyWheel,
			Extra: artifact.Extras{
				artifact.ExtraID: id,
			},
		})
	}

	arts := contextArtifacts(ctx, config.DockerV2{
		Platforms: []string{"linux/arm/v7", "linux/amd64", "linux/arm64"},
		IDs:       []string{"id1"},
	})
	require.Len(t, arts, 4)

	t.Run("no ids", func(t *testing.T) {
		arts := contextArtifacts(ctx, config.DockerV2{
			Platforms: []string{"linux/arm/v7", "linux/amd64", "linux/arm64"},
		})
		require.Len(t, arts, 5)
	})

	t.Run("amd64 variant", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		for _, goamd64 := range []string{"v1", "v3"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "mybin",
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: goamd64,
				Type:    artifact.Binary,
				Extra: artifact.Extras{
					artifact.ExtraID: "id1",
				},
			})
		}

		arts := contextArtifacts(ctx, config.DockerV2{
			Platforms: []string{"linux/amd64/v3"},
			IDs:       []string{"id1"},
		})
		require.Len(t, arts, 1)
		require.Equal(t, "v3", arts[0].Goamd64)
	})

	t.Run("arm64 variant", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:    "mybin",
			Goos:    "linux",
			Goarch:  "arm64",
			Goarm64: "v8.0",
			Type:    artifact.Binary,
			Extra: artifact.Extras{
				artifact.ExtraID: "id1",
			},
		})

		arts := contextArtifacts(ctx, config.DockerV2{
			Platforms: []string{"linux/arm64/v8"},
			IDs:       []string{"id1"},
		})
		require.Len(t, arts, 1)
	})

	t.Run("arm variants", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		for _, goarm := range []string{"5", "6", "7"} {
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "mybin",
				Goos:   "linux",
				Goarch: "arm",
				Goarm:  goarm,
				Type:   artifact.Binary,
				Extra: artifact.Extras{
					artifact.ExtraID: "id1",
				},
			})
		}

		arts := contextArtifacts(ctx, config.DockerV2{
			Platforms: []string{"linux/arm/v5", "linux/arm/v6", "linux/arm/v7"},
			IDs:       []string{"id1"},
		})
		require.Len(t, arts, 3)
	})
}

func TestIsRetriableBuild(t *testing.T) {
	t.Run("retriable", func(t *testing.T) {
		for _, out := range []string{
			"manifest verification failed for digest",
			"toomanyrequests: You have reached your pull rate limit",
			"failed to do request: connection reset by peer",
			"received unexpected HTTP status: 503 Service Unavailable",
			"error pulling image configuration: unexpected EOF",
			"Temporary failure resolving 'deb.debian.org'",
			"E: Failed to fetch http://deb.debian.org/... Connection timed out",
			"WARNING: fetching https://dl-cdn.alpinelinux.org: temporary error (try again later)",
			"context deadline exceeded",
		} {
			t.Run(out, func(t *testing.T) {
				require.True(t, IsRetriableBuild(gerrors.Wrap(errors.New("exit status 1"), gerrors.WithOutput(out))))
			})
		}
	})
	t.Run("not retriable", func(t *testing.T) {
		require.False(t, IsRetriableBuild(gerrors.Wrap(nil, gerrors.WithOutput("some other error"))))
		require.False(t, IsRetriableBuild(gerrors.Wrap(nil, gerrors.WithOutput(">>> COPY failed: file not found"))))
		require.False(t, IsRetriableBuild(errors.New("some other error")))
		require.False(t, IsRetriableBuild(nil))
	})
	t.Run("output embedded in error message", func(t *testing.T) {
		require.True(t, IsRetriableBuild(errors.New("failed to build: toomanyrequests")))
	})
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

func TestIsDriverValid(t *testing.T) {
	require.True(t, isDriverValid("docker-container"))
	require.False(t, isDriverValid("other"))
}

func TestRunHook(t *testing.T) {
	fields := tmpl.Fields{
		keyImages:     []string{"foo/bar:v1", "foo/bar:latest"},
		keyDockerfile: "/path/to/Dockerfile",
		keyDigest:     "sha256:deadbeef",
		keyContextDir: "/tmp/goreleaserdocker123",
	}

	t.Run("no hooks", func(t *testing.T) {
		require.NoError(t, runHook(testctx.Wrap(t.Context()), fields, nil))
	})

	t.Run("simple", func(t *testing.T) {
		dir := t.TempDir()
		marker := filepath.Join(dir, "marker")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runHook(ctx, fields, config.Hooks{
			{Cmd: testlib.Touch(marker)},
		}))
		require.FileExists(t, marker)
	})

	t.Run("with extra fields", func(t *testing.T) {
		dir := t.TempDir()
		out := filepath.Join(dir, "out")
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runHook(ctx, fields, config.Hooks{
			{
				Cmd:    testlib.ShC(`echo "{{.Dockerfile}} {{.Digest}} {{.ContextDir}} {{range .Images}}{{.}} {{end}}" > ` + out),
				Output: true,
			},
		}))
		bts, err := os.ReadFile(out)
		require.NoError(t, err)
		require.Contains(t, string(bts), "/path/to/Dockerfile")
		require.Contains(t, string(bts), "sha256:deadbeef")
		require.Contains(t, string(bts), "/tmp/goreleaserdocker123")
		require.Contains(t, string(bts), "foo/bar:v1")
		require.Contains(t, string(bts), "foo/bar:latest")
	})

	t.Run("env tmpl", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.NoError(t, runHook(ctx, fields, config.Hooks{
			{
				Cmd: testlib.Echo("{{.Env.FOO}}"),
				Env: []string{"FOO=bar-{{.Tag}}"},
			},
		}))
	})

	t.Run("bad env tmpl", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		testlib.RequireTemplateError(t, runHook(ctx, fields, config.Hooks{
			{
				Cmd: testlib.Echo("blah"),
				Env: []string{"FOO=foo-{{.Tag}"},
			},
		}))
	})

	t.Run("bad dir tmpl", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		testlib.RequireTemplateError(t, runHook(ctx, fields, config.Hooks{
			{
				Cmd: testlib.Echo("blah"),
				Dir: "{{.Tag}",
			},
		}))
	})

	t.Run("bad cmd tmpl", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		testlib.RequireTemplateError(t, runHook(ctx, fields, config.Hooks{
			{Cmd: "echo blah-{{.Tag }"},
		}))
	})

	t.Run("unparseable cmd", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		require.Error(t, runHook(ctx, fields, config.Hooks{
			{Cmd: `echo "unterminated`},
		}))
	})

	t.Run("failing cmd", func(t *testing.T) {
		ctx := testctx.Wrap(t.Context())
		err := runHook(ctx, fields, config.Hooks{
			{Cmd: "exit 1"},
		})
		require.ErrorIs(t, err, exec.ErrNotFound)
	})
}
