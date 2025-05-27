package docker

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/stretchr/testify/require"
)

func TestPlatFor(t *testing.T) {
	for expected, art := range map[string]artifact.Artifact{
		"darwin/amd64": {
			Goos:   "darwin",
			Goarch: "amd64",
		},
		"darwin/arm64": {
			Goos:   "darwin",
			Goarch: "arm64",
		},
		"windows/amd64": {
			Goos:   "windows",
			Goarch: "amd64",
		},
		"windows/arm64": {
			Goos:   "windows",
			Goarch: "arm64",
		},
		"linux/amd64": {
			Goos:   "linux",
			Goarch: "amd64",
		},
		"linux/arm64": {
			Goos:   "linux",
			Goarch: "arm64",
		},
		"linux/arm/v7": {
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "7",
		},
		"linux/arm/v6": {
			Goos:   "linux",
			Goarch: "arm",
			Goarm:  "6",
		},
		"linux/386": {
			Goos:   "linux",
			Goarch: "386",
		},
		"linux/ppc64le": {
			Goos:   "linux",
			Goarch: "ppc64le",
		},
		"linux/s390x": {
			Goos:   "linux",
			Goarch: "s390x",
		},
		"linux/riscv64": {
			Goos:   "linux",
			Goarch: "riscv64",
		},
	} {
		t.Run(expected, func(t *testing.T) {
			plat, err := toPlatform(&art)
			require.NoError(t, err)
			require.Equal(t, expected, plat)
		})
	}
}
