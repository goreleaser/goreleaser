package docker

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCommandForBuildPack(t *testing.T) {
	images := []string{"goreleaser/test_build_flag", "goreleaser/test_multiple_tags"}
	tests := []struct {
		name   string
		flags  []string
		expect []string
	}{
		{
			name:   "no flags without builder",
			flags:  []string{},
			expect: []string{"build", images[0], "-t", images[1], "--builder=gcr.io/buildpacks/builder:v1"},
		},
		{
			name:   "single flag without builder",
			flags:  []string{"--clear-cache"},
			expect: []string{"build", images[0], "-t", images[1], "--clear-cache", "--builder=gcr.io/buildpacks/builder:v1"},
		},
		{
			name:   "multiple flags without builder",
			flags:  []string{"--clear-cache", "--verbose"},
			expect: []string{"build", images[0], "-t", images[1], "--clear-cache", "--verbose", "--builder=gcr.io/buildpacks/builder:v1"},
		},
		{
			name:   "builder with --builder flag",
			flags:  []string{"--builder=heroku/buildpacks:20"},
			expect: []string{"build", images[0], "-t", images[1], "--builder=heroku/buildpacks:20"},
		},
		{
			name:   "builder with -B flag",
			flags:  []string{"-B=heroku/buildpacks:18"},
			expect: []string{"build", images[0], "-t", images[1], "-B=heroku/buildpacks:18"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imager := buildPackImager{}
			require.Equal(t, tt.expect, imager.buildCommand(images, tt.flags))
		})
	}
}
