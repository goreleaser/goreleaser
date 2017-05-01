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
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}

	assert.NoError(Pipe{}.Run(ctx))

	assert.Equal("goreleaser", ctx.Config.Release.GitHub.Owner)
	assert.Equal("goreleaser", ctx.Config.Release.GitHub.Name)
	assert.Equal("goreleaser", ctx.Config.Build.Binary)
	assert.Equal(".", ctx.Config.Build.Main)
	assert.Equal("tar.gz", ctx.Config.Archive.Format)
	assert.Contains(ctx.Config.Build.Goos, "darwin")
	assert.Contains(ctx.Config.Build.Goos, "linux")
	assert.Contains(ctx.Config.Build.Goarch, "386")
	assert.Contains(ctx.Config.Build.Goarch, "amd64")
	assert.Contains(ctx.Config.Brew.Install, "bin.install \"goreleaser\"")
	assert.NotEmpty(
		ctx.Config.Archive.Replacements,
		ctx.Config.Archive.NameTemplate,
		ctx.Config.Build.Ldflags,
		ctx.Config.Archive.Files,
	)
}

func TestFillPartial(t *testing.T) {
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{
			Release: config.Release{
				GitHub: config.Repo{
					Owner: "goreleaser",
					Name:  "test",
				},
			},
		},
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestFilesFilled(t *testing.T) {
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				Files: []string{
					"README.md",
				},
			},
		},
	}

	assert.NoError(Pipe{}.Run(ctx))
	assert.Len(ctx.Config.Archive.Files, 1)
}

func TestAcceptFiles(t *testing.T) {
	var files = []string{
		"LICENSE.md",
		"LIceNSE.txt",
		"LICENSE",
		"LICENCE.txt",
		"LICEncE",
		"README",
		"READme.md",
		"CHANGELOG.txt",
		"ChanGELOG.md",
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			assert.True(t, accept(file))
		})
	}
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
