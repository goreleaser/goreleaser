package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/stretchr/testify/assert"
)

func TestExtWindows(t *testing.T) {
	assert.Equal(t, ".exe", extFor("windows"))
}

func TestExtOthers(t *testing.T) {
	assert.Empty(t, "", extFor("linux"))
}

func TestNameFor(t *testing.T) {
	assert := assert.New(t)

	var config = config.Project{
		Archive: config.Archive{
			NameTemplate: "{{.Binary}}_{{.Os}}_{{.Arch}}_{{.Tag}}_{{.Version}}",
			Replacements: map[string]string{
				"darwin": "Darwin",
				"amd64":  "x86_64",
			},
		},
		Build: config.Build{
			Binary: "test",
		},
	}
	var ctx = &context.Context{
		Config:  config,
		Version: "1.2.3",
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	name, err := nameFor(ctx, buildTarget{"darwin", "amd64", ""})
	assert.NoError(err)
	assert.Equal("test_Darwin_x86_64_v1.2.3_1.2.3", name)
}

func TestInvalidNameTemplate(t *testing.T) {
	assert := assert.New(t)

	var config = config.Project{
		Archive: config.Archive{
			NameTemplate: "{{.Binary}_{{.Os}}_{{.Arch}}_{{.Version}}",
		},
		Build: config.Build{
			Binary: "test",
		},
	}
	var ctx = &context.Context{
		Config: config,
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	_, err := nameFor(ctx, buildTarget{"darwin", "amd64", ""})
	assert.Error(err)
}

func TestNameDefaltTemplate(t *testing.T) {
	assert := assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				NameTemplate: defaults.NameTemplate,
			},
			Build: config.Build{
				Binary: "test",
			},
		},
		Version: "1.2.3",
	}
	for key, target := range map[string]buildTarget{
		"test_darwin_amd64": buildTarget{"darwin", "amd64", ""},
		"test_linux_arm64":  buildTarget{"linux", "arm64", ""},
		"test_linux_armv7":  buildTarget{"linux", "arm", "7"},
	} {
		t.Run(key, func(t *testing.T) {
			name, err := nameFor(ctx, target)
			assert.NoError(err)
			assert.Equal(key, name)
		})
	}

}
