package build

import (
	"os"
	"path/filepath"
	"testing"

	api "github.com/goreleaser/goreleaser/build"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/stretchr/testify/assert"
)

type fakeBuilder struct{}

func (*fakeBuilder) Default(build config.Build) config.Build {
	return build
}

func (*fakeBuilder) Build(ctx *context.Context, build config.Build, options api.Options) error {
	ctx.Artifacts.Add(artifact.Artifact{})
	return nil
}

func init() {
	api.Register("fake", &fakeBuilder{})
}

func TestPipeDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestBuild(t *testing.T) {
	var builder = &fakeBuilder{}
	api.Register("fake", builder)
	var config = config.Project{
		Builds: []config.Build{
			{
				Lang:   "fake",
				Binary: "testing",
				Flags:  "-n",
				Env:    []string{"BLAH=1"},
			},
		},
	}
	var ctx = context.New(config)
	assert.NoError(t, doBuild(ctx, ctx.Config.Builds[0], "darwin_amd64"))
}

func TestRunFullPipe(t *testing.T) {
	folder, back := testlib.Mktmp(t)
	defer back()
	var pre = filepath.Join(folder, "pre")
	var post = filepath.Join(folder, "post")
	var config = config.Project{
		Builds: []config.Build{
			{
				Lang:    "fake",
				Binary:  "testing",
				Flags:   "-v",
				Ldflags: "-X main.test=testing",
				Hooks: config.Hooks{
					Pre:  "touch " + pre,
					Post: "touch " + post,
				},
				Targets: []string{"whatever"},
			},
		},
		Archive: config.Archive{
			Replacements: map[string]string{
				"linux":  "linuxx",
				"darwin": "darwinn",
			},
		},
	}
	var ctx = context.New(config)
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.List(), 1)
	assert.True(t, exists(pre), pre)
	assert.True(t, exists(post), post)
}

func TestRunPipeFailingHooks(t *testing.T) {
	var config = config.Project{
		Builds: []config.Build{
			{
				Lang:    "fake",
				Binary:  "hooks",
				Hooks:   config.Hooks{},
				Targets: []string{"whatever"},
			},
		},
	}
	t.Run("pre-hook", func(t *testing.T) {
		var ctx = context.New(config)
		ctx.Config.Builds[0].Hooks.Pre = "exit 1"
		ctx.Config.Builds[0].Hooks.Post = "echo post"
		assert.EqualError(t, Pipe{}.Run(ctx), `pre hook failed: `)
	})
	t.Run("post-hook", func(t *testing.T) {
		var ctx = context.New(config)
		ctx.Config.Builds[0].Hooks.Pre = "echo pre"
		ctx.Config.Builds[0].Hooks.Post = "exit 1"
		assert.EqualError(t, Pipe{}.Run(ctx), `post hook failed: `)
	})
}

func TestDefaultNoBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultEmptyBuild(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name: "foo",
				},
			},
			Builds: []config.Build{
				{},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	var build = ctx.Config.Builds[0]
	assert.Equal(t, ctx.Config.Release.GitHub.Name, build.Binary)
	assert.Equal(t, ".", build.Main)
	assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
	assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
	assert.Equal(t, []string{"6"}, build.Goarm)
	assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}", build.Ldflags)
}

func TestDefaultPartialBuilds(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					Binary: "bar",
					Goos:   []string{"linux"},
					Main:   "./cmd/main.go",
				},
				{
					Binary:  "foo",
					Ldflags: "-s -w",
					Goarch:  []string{"386"},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	t.Run("build0", func(t *testing.T) {
		var build = ctx.Config.Builds[0]
		assert.Equal(t, "bar", build.Binary)
		assert.Equal(t, "./cmd/main.go", build.Main)
		assert.Equal(t, []string{"linux"}, build.Goos)
		assert.Equal(t, []string{"amd64", "386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Equal(t, "-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}", build.Ldflags)
	})
	t.Run("build1", func(t *testing.T) {
		var build = ctx.Config.Builds[1]
		assert.Equal(t, "foo", build.Binary)
		assert.Equal(t, ".", build.Main)
		assert.Equal(t, []string{"linux", "darwin"}, build.Goos)
		assert.Equal(t, []string{"386"}, build.Goarch)
		assert.Equal(t, []string{"6"}, build.Goarm)
		assert.Equal(t, "-s -w", build.Ldflags)
	})
}

func TestDefaultFillSingleBuild(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()

	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Name: "foo",
				},
			},
			SingleBuild: config.Build{
				Main: "testreleaser",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Builds, 1)
	assert.Equal(t, ctx.Config.Builds[0].Binary, "foo")
}

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ".exe", extFor("windows_amd64"))
	assert.Equal(t, ".exe", extFor("windows_386"))
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, "", extFor("linux_amd64"))
	assert.Empty(t, "", extFor("linuxwin_386"))
	assert.Empty(t, "", extFor("winasdasd_sad"))
}

//
// Helpers
//

func exists(file string) bool {
	_, err := os.Stat(file)
	return !os.IsNotExist(err)
}
