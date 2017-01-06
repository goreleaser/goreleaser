package name

import (
	"testing"

	"github.com/goreleaser/releaser/config"
	"github.com/stretchr/testify/assert"
)

func TestNameTemplate(t *testing.T) {
	assert := assert.New(t)
	var config = config.ProjectConfig{
		BinaryName:   "test",
		NameTemplate: "{{.BinaryName}}_{{.Os}}_{{.Arch}}_{{.Version}}",
		Git: config.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}
	name, err := For(config, "darwin", "amd64")
	assert.NoError(err)
	assert.Equal("test_Darwin_x86_64_v1.2.3", name)
}
