package defaults

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestFillBasicData(t *testing.T) {
	assert := assert.New(t)

	var config = &config.ProjectConfig{}
	var ctx = &context.Context{
		Config: config,
	}

	assert.NoError(Pipe{}.Run(ctx))

	assert.Equal("goreleaser/releaser", config.Repo)
	assert.Equal("releaser", config.BinaryName)
	assert.Equal("main.go", config.Build.Main)
	assert.Equal("tar.gz", config.Archive.Format)
	assert.Contains(config.Build.Goos, "darwin")
	assert.Contains(config.Build.Goos, "linux")
	assert.Contains(config.Build.Goarch, "386")
	assert.Contains(config.Build.Goarch, "amd64")
	assert.NotEmpty(
		config.Archive.Replacements,
		config.Archive.NameTemplate,
		config.Build.Ldflags,
		config.Archive.Files,
	)
}

func TestFilesFilled(t *testing.T) {
	assert := assert.New(t)

	var config = &config.ProjectConfig{
		Archive: config.ArchiveConfig{
			Files: []string{
				"README.md",
			},
		},
	}
	var ctx = &context.Context{
		Config: config,
	}

	assert.NoError(Pipe{}.Run(ctx))
	assert.Len(config.Archive.Files, 1)
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
