package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestAllBuildTargets(t *testing.T) {
	var assert = assert.New(t)
	var ctx = &context.Context{
		Config: config.Project{
			Build: config.Build{
				Goos: []string{
					"linux",
					"darwin",
					"freebsd",
				},
				Goarch: []string{
					"386",
					"amd64",
					"arm",
					"arm64",
				},
				Goarm: []string{
					"6",
					"7",
				},
				Ignore: []config.IgnoredBuild{
					{
						Goos:   "darwin",
						Goarch: "386",
					}, {
						Goos:   "linux",
						Goarch: "arm",
						Goarm:  "7",
					},
				},
			},
		},
	}
	assert.Equal([]buildTarget{
		buildTarget{"linux", "386", ""},
		buildTarget{"linux", "amd64", ""},
		buildTarget{"linux", "arm", "6"},
		buildTarget{"linux", "arm64", ""},
		buildTarget{"darwin", "amd64", ""},
		buildTarget{"freebsd", "386", ""},
		buildTarget{"freebsd", "amd64", ""},
		buildTarget{"freebsd", "arm", "6"},
		buildTarget{"freebsd", "arm", "7"},
	}, allBuildTargets(ctx))
}
