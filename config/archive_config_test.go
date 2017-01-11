package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameTemplate(t *testing.T) {
	assert := assert.New(t)
	var config = ProjectConfig{
		BinaryName: "test",
		Git: GitInfo{
			CurrentTag: "v1.2.3",
		},
		Archive: ArchiveConfig{
			NameTemplate: "{{.BinaryName}}_{{.Os}}_{{.Arch}}_{{.Version}}",
			Replacements: map[string]string{
				"darwin": "Darwin",
				"amd64":  "x86_64",
			},
		},
	}
	name, err := config.ArchiveName("darwin", "amd64")
	assert.NoError(err)
	assert.Equal("test_Darwin_x86_64_v1.2.3", name)
}
