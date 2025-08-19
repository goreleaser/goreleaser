package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	dist := t.TempDir()
	binpath := filepath.Join(dist, "mybin")
	require.NoError(t, os.WriteFile(binpath, []byte("#!/bin/sh\necho hi"), 0o755))
	require.NoError(t, gio.Copy("./testdata/Dockerfile", filepath.Join(dist, "Dockerfile")))
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "dockerv2",
		Dist:        dist,
		DockersV2: []config.DockerV2{
			{
				ID:         "myimg",
				Dockerfile: "./testdata/Dockerfile",
				Images:     []string{"image1", "image2"},
				Tags:       []string{"tag1", "tag2"},
				Files:      []string{"./testdata/foo.conf"},
				// IDs:        []string{"id1"},
			},
		},
	}, testctx.Snapshot)
	for _, arch := range []string{"amd64", "arm64"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Path:   binpath,
			Goos:   "linux",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: artifact.Extras{
				artifact.ExtraID: "id1",
			},
		})
	}

	require.NoError(t, Pipe{}.Default(ctx))
	err := Pipe{}.Run(ctx)
	require.NoError(t, err, "message: %s, output: %v", gerrors.MessageOf(err), gerrors.DetailsOf(err))

	images := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImageV2)).List()
	require.Len(t, images, 2)
	t.Log(images)
	require.Regexp(t, `localhost:\d+/dockerv2/myimg:tag1`, images[0])
	require.Regexp(t, `localhost:\d+/dockerv2/myimg:tag2`, images[1])
}
