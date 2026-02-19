package docker

import (
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestBuildCommand(t *testing.T) {
	images := []string{"goreleaser/test_build_flag", "goreleaser/test_multiple_tags"}
	tests := []struct {
		name   string
		flags  []string
		buildx bool
		expect []string
	}{
		{
			name:   "no flags",
			flags:  []string{},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--provenance=false", "--sbom=false"},
		},
		{
			name:   "single flag",
			flags:  []string{"--label=foo"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo", "--provenance=false", "--sbom=false"},
		},
		{
			name:   "multiple flags",
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz", "--provenance=false", "--sbom=false"},
		},
		{
			name:   "buildx",
			buildx: true,
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"buildx", "build", ".", "--load", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz", "--provenance=false", "--sbom=false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imager := dockerImager{
				buildx: tt.buildx,
			}
			require.Equal(t, tt.expect, imager.buildCommand(images, tt.flags))
		})
	}
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNoDockerWithoutImageName(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Run(testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	})))
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				IDs: []string{"aa"},
			},
			{
				Use: useBuildx,
			},
		},
		DockerManifests: []config.DockerManifest{
			{},
			{
				Use: useDocker,
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "linux", docker.Goos)
	require.Equal(t, "amd64", docker.Goarch)
	require.Equal(t, "6", docker.Goarm)
	require.Equal(t, []string{"aa"}, docker.IDs)
	require.Equal(t, useDocker, docker.Use)
	docker = ctx.Config.Dockers[1]
	require.Equal(t, useBuildx, docker.Use)
	require.Equal(t, uint(10), docker.Retry.Attempts)
	require.Equal(t, 10*time.Second, docker.Retry.Delay)
	require.Equal(t, 5*time.Minute, docker.Retry.MaxDelay)

	require.NoError(t, ManifestPipe{}.Default(ctx))
	require.Len(t, ctx.Config.DockerManifests, 2)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[0].Use)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[1].Use)

	for _, manifest := range ctx.Config.DockerManifests {
		require.Equal(t, uint(10), manifest.Retry.Attempts)
		require.Equal(t, 10*time.Second, manifest.Retry.Delay)
		require.Equal(t, 5*time.Minute, manifest.Retry.MaxDelay)
	}
}

func TestDefaultDuplicateID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{ID: "foo"},
			{},
			{ID: "bar"},
			{ID: "foo"},
		},
		DockerManifests: []config.DockerManifest{
			{ID: "bar"},
			{},
			{ID: "bar"},
			{ID: "foo"},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 dockers with the ID 'foo', please fix your config")
	require.EqualError(t, ManifestPipe{}.Default(ctx), "found 2 docker_manifests with the ID 'bar', please fix your config")
}

func TestDefaultInvalidUse(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				Use: "something",
			},
		},
		DockerManifests: []config.DockerManifest{
			{
				Use: "something",
			},
		},
	})

	err := Pipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker: invalid use: something, valid options are`))

	err = ManifestPipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker manifest: invalid use: something, valid options are`))
}

func TestDefaultDockerfile(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{},
		},
		Dockers: []config.Docker{
			{},
			{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[0].Dockerfile)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[1].Dockerfile)
}

func TestDraftRelease(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			Draft: true,
		},
	})

	require.False(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestDefaultNoDockers(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Dockers)
}

func TestDefaultFilesDot(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "/tmp/distt",
		Dockers: []config.Docker{
			{
				Files: []string{"./lala", "./lolsob", "."},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultFilesDis(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "/tmp/dist",
		Dockers: []config.Docker{
			{
				Files: []string{"./fooo", "/tmp/dist/asdasd/asd", "./bar"},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				IDs:        []string{"foo"},
				Goos:       "windows",
				Goarch:     "i386",
				Dockerfile: "Dockerfile.foo",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 1)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "windows", docker.Goos)
	require.Equal(t, "i386", docker.Goarch)
	require.Equal(t, []string{"foo"}, docker.IDs)
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)
}

func Test_processImageTemplates(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Builds: []config.Build{
				{
					ID: "default",
				},
			},
			Dockers: []config.Docker{
				{
					Dockerfile: "Dockerfile.foo",
					ImageTemplates: []string{
						"user/image:{{.Tag}}",
						"gcr.io/image:{{.Tag}}-{{.Env.FOO}}",
						"gcr.io/image:v{{.Major}}.{{.Minor}}",
					},
					SkipPush: "true",
				},
			},
			Env: []string{"FOO=123"},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithCommit("a1b2c3d4"),
		testctx.WithSemver(1, 0, 0, ""))

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 1)

	docker := ctx.Config.Dockers[0]
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)

	images, err := processImageTemplates(ctx, docker)
	require.NoError(t, err)
	require.Equal(t, []string{
		"user/image:v1.0.0",
		"gcr.io/image:v1.0.0-123",
		"gcr.io/image:v1.0",
	}, images)
}

func TestSkip(t *testing.T) {
	t.Run("image", func(t *testing.T) {
		t.Run("skip", func(t *testing.T) {
			require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
		})

		t.Run("skip docker", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dockers: []config.Docker{{}},
			}, testctx.Skip(skips.Docker))

			require.True(t, Pipe{}.Skip(ctx))
		})

		t.Run("dont skip", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dockers: []config.Docker{{}},
			})

			require.False(t, Pipe{}.Skip(ctx))
		})
	})

	t.Run("manifest", func(t *testing.T) {
		t.Run("skip", func(t *testing.T) {
			require.True(t, ManifestPipe{}.Skip(testctx.Wrap(t.Context())))
		})

		t.Run("skip docker", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				DockerManifests: []config.DockerManifest{{}},
			}, testctx.Skip(skips.Docker))

			require.True(t, ManifestPipe{}.Skip(ctx))
		})

		t.Run("dont skip", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				DockerManifests: []config.DockerManifest{{}},
			})

			require.False(t, ManifestPipe{}.Skip(ctx))
		})
	})
}

func TestWithDigest(t *testing.T) {
	artifacts := artifact.New()
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t1",
		Type: artifact.DockerImage,
		Extra: artifact.Extras{
			artifact.ExtraDigest: "sha256:d1",
		},
	})
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t2",
		Type: artifact.DockerImage,
		Extra: artifact.Extras{
			artifact.ExtraDigest: "sha256:d2",
		},
	})
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t3",
		Type: artifact.DockerImage,
	})

	for _, use := range []string{useDocker, useBuildx} {
		t.Run(use, func(t *testing.T) {
			t.Run("good", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t1@sha256:d1", withDigest("localhost:5050/owner/img:t1", artifacts.List()))
			})

			t.Run("no digest", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t3", withDigest("localhost:5050/owner/img:t3", artifacts.List()))
			})

			t.Run("no match", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t4", withDigest("localhost:5050/owner/img:t4", artifacts.List()))
			})
		})
	}
}

func TestDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{Use: useBuildx},
			{Use: useDocker},
			{Use: "nope"},
		},
		DockerManifests: []config.DockerManifest{
			{Use: useBuildx},
			{Use: useDocker},
			{Use: "nope"},
		},
	})

	require.Equal(t, []string{"docker", "docker"}, Pipe{}.Dependencies(ctx))
	require.Equal(t, []string{"docker", "docker"}, ManifestPipe{}.Dependencies(ctx))
}

func TestIsFileNotFoundError(t *testing.T) {
	t.Run("executable not in path", func(t *testing.T) {
		require.False(t, isFileNotFoundError(`error getting credentials - err: exec: "docker-credential-desktop": executable file not found in $PATH, out:`))
	})

	t.Run("file not found", func(t *testing.T) {
		require.True(t, isFileNotFoundError(`./foo: file not found`))
		require.True(t, isFileNotFoundError(`./foo: not found: not found`))
	})
}

func TestValidateImager(t *testing.T) {
	tests := []struct {
		use       string
		wantError string
	}{
		{use: "docker"},
		{use: "buildx"},
		{use: "notFound", wantError: "docker: invalid use: notFound, valid options are [buildx docker]"},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			err := validateImager(tt.use)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
