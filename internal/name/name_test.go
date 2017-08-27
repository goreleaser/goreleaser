package name

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/buildtarget"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/stretchr/testify/assert"
)

func TestChecksums(t *testing.T) {
	var assert = assert.New(t)

	var config = config.Project{
		Checksum: config.Checksum{
			NameTemplate: "{{.ProjectName }}_{{.Tag}}_{{.Version}}_checksums.txt",
		},
		ProjectName: "testcheck",
	}
	var ctx = &context.Context{
		Config:  config,
		Version: "1.0.0",
		Git: context.GitInfo{
			CurrentTag: "v1.0.0",
		},
	}

	name, err := ForChecksums(ctx)
	assert.NoError(err)
	assert.Equal("testcheck_v1.0.0_1.0.0_checksums.txt", name)
}

func TestNameFor(t *testing.T) {
	var assert = assert.New(t)

	var config = config.Project{
		Archive: config.Archive{
			NameTemplate: "{{.Binary}}_{{.Os}}_{{.Arch}}_{{.Tag}}_{{.Version}}",
			Replacements: map[string]string{
				"darwin": "Darwin",
				"amd64":  "x86_64",
			},
		},
		ProjectName: "test",
	}
	var ctx = &context.Context{
		Config:  config,
		Version: "1.2.3",
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	name, err := For(ctx, buildtarget.New("darwin", "amd64", ""))
	assert.NoError(err)
	assert.Equal("test_Darwin_x86_64_v1.2.3_1.2.3", name)
}

func TestNameForBuild(t *testing.T) {
	var assert = assert.New(t)

	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				NameTemplate: "{{.Binary}}_{{.Os}}_{{.Arch}}_{{.Tag}}_{{.Version}}",
				Replacements: map[string]string{
					"darwin": "Darwin",
					"amd64":  "x86_64",
				},
			},
			ProjectName: "test",
		},
		Version: "1.2.3",
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	name, err := ForBuild(
		ctx,
		config.Build{Binary: "foo"},
		buildtarget.New("darwin", "amd64", ""),
	)
	assert.NoError(err)
	assert.Equal("foo_Darwin_x86_64_v1.2.3_1.2.3", name)
}

func TestInvalidNameTemplate(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				NameTemplate: "{{.Binary}_{{.Os}}_{{.Arch}}_{{.Version}}",
			},
			ProjectName: "test",
		},
		Git: context.GitInfo{
			CurrentTag: "v1.2.3",
		},
	}

	_, err := For(ctx, buildtarget.New("darwin", "amd64", ""))
	assert.Error(err)
}

func TestNameDefaultTemplate(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{
			Archive: config.Archive{
				NameTemplate: defaults.NameTemplate,
			},
			ProjectName: "test",
		},
		Version: "1.2.3",
	}
	type buildTarget struct {
		goos, goarch, goarm string
	}
	for key, target := range map[string]buildtarget.Target{
		"test_1.2.3_darwin_amd64": buildtarget.New("darwin", "amd64", ""),
		"test_1.2.3_linux_arm64":  buildtarget.New("linux", "arm64", ""),
		"test_1.2.3_linux_armv7":  buildtarget.New("linux", "arm", "7"),
	} {
		t.Run(key, func(t *testing.T) {
			name, err := For(ctx, target)
			assert.NoError(err)
			assert.Equal(key, name)
		})
	}
}
