package release

import (
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDescribeBody(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := context.New(config.Project{})
	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/goreleaser",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDescribeBodyWithDockerManifest(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := context.New(config.Project{})
	ctx.ReleaseNotes = changelog
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0",
		"goreleaser/goreleaser:latest",
		"goreleaser/godownloader:v0.1.0",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerManifest,
		})
	}
	for _, d := range []string{
		"goreleaser/goreleaser:0.40.0-amd64",
		"goreleaser/goreleaser:latest-amd64",
		"goreleaser/godownloader:v0.1.0-amd64",
		"goreleaser/goreleaser:0.40.0-arm64",
		"goreleaser/goreleaser:latest-arm64",
		"goreleaser/godownloader:v0.1.0-arm64",
	} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: d,
			Type: artifact.DockerImage,
		})
	}
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDescribeBodyNoDockerImagesNoBrews(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := &context.Context{
		ReleaseNotes: changelog,
	}
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}

func TestDontEscapeHTML(t *testing.T) {
	changelog := "<h1>test</h1>"
	ctx := context.New(config.Project{})
	ctx.ReleaseNotes = changelog

	out, err := describeBody(ctx)
	require.NoError(t, err)
	require.Contains(t, out.String(), changelog)
}

func TestDescribeBodyWithHeaderAndFooter(t *testing.T) {
	changelog := "feature1: description\nfeature2: other description"
	ctx := context.New(config.Project{
		Release: config.Release{
			Header: "## Yada yada yada\nsomething\n",
			Footer: "\n---\n\nGet images at docker.io/foo/bar:{{.Tag}}\n\n---\n\nGet GoReleaser Pro at https://goreleaser.com/pro",
		},
	})
	ctx.ReleaseNotes = changelog
	ctx.Git = context.GitInfo{CurrentTag: "v1.0"}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "goreleaser/goreleaser:v1.2.3",
		Type: artifact.DockerImage,
	})
	out, err := describeBody(ctx)
	require.NoError(t, err)

	golden.RequireEqual(t, out.Bytes())
}
