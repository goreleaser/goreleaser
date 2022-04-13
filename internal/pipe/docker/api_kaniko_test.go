package docker

import (
	"fmt"
	"github.com/goreleaser/goreleaser/pkg/config"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKanikoImagerBuildCommand(t *testing.T) {
	images := []string{"goreleaser/test_build_flag", "goreleaser/test_multiple_tags"}
	cwd, _ := os.Getwd()
	tests := []struct {
		name       string
		flags      []string
		dockerfile string
		skipPush   string
		expect     []string
	}{
		{
			name:       "kaniko build on docker",
			flags:      []string{"--label=foo=bar", "--build-arg=bar=baz"},
			dockerfile: "Dockerfile",
			expect:     []string{"run", "-v", fmt.Sprintf("%s:/workspace", cwd), kanikoExecutorImage, "--context", "dir:///workspace", "--dockerfile", "/workspace/Dockerfile", "--destination", images[0], "--destination", images[1], "--label=foo=bar", "--build-arg=bar=baz"},
		},
		{
			name:       "kaniko build with skip push",
			dockerfile: "test/Dockerfile",
			skipPush:   "true",
			expect:     []string{"run", "-v", fmt.Sprintf("%s:/workspace", cwd), kanikoExecutorImage, "--context", "dir:///workspace", "--dockerfile", "/workspace/test/Dockerfile", "--destination", images[0], "--destination", images[1], "--no-push"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imager := kanikoImager{}
			docker := config.Docker{Dockerfile: tt.dockerfile, SkipPush: tt.skipPush}
			require.Equal(t, tt.expect, imager.buildCommand(docker, images, tt.flags))
		})
	}
}
