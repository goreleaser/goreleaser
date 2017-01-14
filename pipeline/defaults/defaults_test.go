package defaults

import (
	"testing"

	"github.com/goreleaser/releaser/config"
	"github.com/goreleaser/releaser/context"
	"github.com/stretchr/testify/assert"
)

func TestFillBasicData(t *testing.T) {
	assert := assert.New(t)

	var config = &config.ProjectConfig{}
	var ctx = &context.Context{
		Config: config,
	}

	assert.NoError(Pipe{}.Run(ctx))

	assert.Equal("main.go", config.Build.Main)
	assert.Equal("tar.gz", config.Archive.Format)
	assert.Contains(config.Build.Oses, "darwin")
	assert.Contains(config.Build.Oses, "linux")
	assert.Contains(config.Build.Arches, "386")
	assert.Contains(config.Build.Arches, "amd64")
}
