package defaults

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestFillBasicData(t *testing.T) {
	assert := assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{},
	}

	assert.NoError(Pipe{}.Run(ctx))

	assert.Equal("goreleaser/goreleaser", ctx.Config.Release.Repo)
	assert.Equal("goreleaser", ctx.Config.Build.BinaryName)
	assert.Equal("main.go", ctx.Config.Build.Main)
	assert.Equal("tar.gz", ctx.Config.Archive.Format)
	assert.Contains(ctx.Config.Build.Goos, "darwin")
	assert.Contains(ctx.Config.Build.Goos, "linux")
	assert.Contains(ctx.Config.Build.Goarch, "386")
	assert.Contains(ctx.Config.Build.Goarch, "amd64")
	assert.NotEmpty(
		ctx.Config.Archive.Replacements,
		ctx.Config.Archive.NameTemplate,
		ctx.Config.Build.Ldflags,
		ctx.Config.Archive.Files,
	)
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
	assert := assert.New(t)

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
		assert.True(accept(file))
	}
}
