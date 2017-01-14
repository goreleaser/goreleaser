package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, extFor("windows"), ".exe")
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, extFor("linux"))
}

func TestNameFor(t *testing.T) {
	assert := assert.New(t)

	var config = &config.ProjectConfig{
		BinaryName: "test",
		Archive: config.ArchiveConfig{
			NameTemplate: "{{.BinaryName}}_{{.Os}}_{{.Arch}}_{{.Version}}",
			Replacements: map[string]string{
				"darwin": "Darwin",
				"amd64":  "x86_64",
			},
		},
	}
	var ctx = &context.Context{
		Config: config,
		Git: &context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	name, err := nameFor(ctx, "darwin", "amd64")
	assert.NoError(err)
	assert.Equal("test_Darwin_x86_64_v1.2.3", name)
}
