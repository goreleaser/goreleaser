package defaults

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestFillBasicData(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{},
	}

	assert.NoError(Pipe{}.Run(ctx))
	assert.Equal("goreleaser", ctx.Config.Release.GitHub.Owner)
	assert.Equal("goreleaser", ctx.Config.Release.GitHub.Name)
	assert.NotEmpty(ctx.Config.Builds)
	assert.Equal("goreleaser", ctx.Config.Builds[0].Binary)
	assert.Equal(".", ctx.Config.Builds[0].Main)
	assert.Contains(ctx.Config.Builds[0].Goos, "darwin")
	assert.Contains(ctx.Config.Builds[0].Goos, "linux")
	assert.Contains(ctx.Config.Builds[0].Goarch, "386")
	assert.Contains(ctx.Config.Builds[0].Goarch, "amd64")
	assert.Equal("tar.gz", ctx.Config.Archive.Format)
	assert.Contains(ctx.Config.Brew.Install, "bin.install \"goreleaser\"")
	assert.Contains(ctx.Config.Brew.CommitAuthor.Name, "goreleaserbot")
	assert.Contains(ctx.Config.Brew.CommitAuthor.Email, "goreleaser@carlosbecker.com")
	assert.Empty(ctx.Config.Dockers)
	assert.NotEmpty(
		ctx.Config.Archive.NameTemplate,
		ctx.Config.Builds[0].Ldflags,
		ctx.Config.Archive.Files,
	)
}

func TestFillPartial(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
			Archive: config.Archive{
				Files: []string{
					"glob/*",
				},
			},
			Builds: []config.Build{
				{Binary: "testreleaser"},
				{Goos: []string{"linux"}},
				{
					Binary: "another",
					Ignore: []config.IgnoredBuild{
						{Goos: "darwin", Goarch: "amd64"},
					},
				},
			},
			Dockers: []config.Docker{
				{Image: "a/b"},
			},
		},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Len(ctx.Config.Archive.Files, 1)
	assert.Equal(`bin.install "testreleaser"`, ctx.Config.Brew.Install)
	assert.NotEmpty(ctx.Config.Dockers[0].Binary)
	assert.NotEmpty(ctx.Config.Dockers[0].Goos)
	assert.NotEmpty(ctx.Config.Dockers[0].Goarch)
	assert.NotEmpty(ctx.Config.Dockers[0].Dockerfile)
	assert.Empty(ctx.Config.Dockers[0].Goarm)
}

func TestFillSingleBuild(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{
			SingleBuild: config.Build{
				Main: "testreleaser",
			},
		},
	}
	assert.NoError(Pipe{}.Run(ctx))
	assert.Len(ctx.Config.Builds, 1)
	assert.Equal(ctx.Config.Builds[0].Binary, "goreleaser")
}

func TestNotAGitRepo(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	previous, err := os.Getwd()
	assert.NoError(err)
	assert.NoError(os.Chdir(folder))
	defer func() {
		assert.NoError(os.Chdir(previous))
	}()
	var ctx = &context.Context{
		Config: config.Project{},
	}
	assert.Error(Pipe{}.Run(ctx))
}
