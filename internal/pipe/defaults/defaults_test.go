package defaults

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestFillBasicData(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = &context.Context{
		Config: config.Project{},
	}

	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Owner)
	assert.Equal(t, "goreleaser", ctx.Config.Release.GitHub.Name)
	assert.NotEmpty(t, ctx.Config.Builds)
	assert.Equal(t, "goreleaser", ctx.Config.Builds[0].Binary)
	assert.Equal(t, ".", ctx.Config.Builds[0].Main)
	assert.Contains(t, ctx.Config.Builds[0].Goos, "darwin")
	assert.Contains(t, ctx.Config.Builds[0].Goos, "linux")
	assert.Contains(t, ctx.Config.Builds[0].Goarch, "386")
	assert.Contains(t, ctx.Config.Builds[0].Goarch, "amd64")
	assert.Equal(t, "tar.gz", ctx.Config.Archives[0].Format)
	assert.Contains(t, ctx.Config.Brew.Install, "bin.install \"goreleaser\"")
	assert.Empty(t, ctx.Config.Dockers)
	assert.Equal(t, "https://github.com", ctx.Config.GitHubURLs.Download)
	assert.NotEmpty(t, ctx.Config.Archives[0].NameTemplate)
	assert.NotEmpty(t, ctx.Config.Builds[0].Ldflags)
	assert.NotEmpty(t, ctx.Config.Archives[0].Files)
	assert.NotEmpty(t, ctx.Config.Dist)
}

func TestFillPartial(t *testing.T) {
	_, back := testlib.Mktmp(t)
	defer back()
	testlib.GitInit(t)
	testlib.GitRemoteAdd(t, "git@github.com:goreleaser/goreleaser.git")

	var ctx = &context.Context{
		Config: config.Project{
			GitHubURLs: config.GitHubURLs{
				Download: "https://github.company.com",
			},
			Dist: "disttt",
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
				{
					ImageTemplates: []string{"a/b"},
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Config.Archive.Files, 1)
	assert.Equal(t, `bin.install "testreleaser"`, ctx.Config.Brew.Install)
	assert.NotEmpty(t, ctx.Config.Dockers[0].Binaries)
	assert.NotEmpty(t, ctx.Config.Dockers[0].Goos)
	assert.NotEmpty(t, ctx.Config.Dockers[0].Goarch)
	assert.NotEmpty(t, ctx.Config.Dockers[0].Dockerfile)
	assert.Empty(t, ctx.Config.Dockers[0].Goarm)
	assert.Equal(t, "disttt", ctx.Config.Dist)
	assert.NotEqual(t, "https://github.com", ctx.Config.GitHubURLs.Download)
}
