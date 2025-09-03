package docker

import (
	"os/exec"
	"slices"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "registry images only available for windows")

	dist := t.TempDir()
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "dockerv2",
		Dist:        dist,
		DockersV2: []config.DockerV2{
			{
				ID:         "myimg",
				Dockerfile: "./testdata/Dockerfile",
				Images:     []string{"image1", "image2"},
				Tags:       []string{"tag1", "tag2"},
				ExtraFiles: []string{"./testdata/foo.conf"},
				IDs:        []string{"id1"},
				Platforms:  []string{"linux/amd64", "linux/arm64", "linux/arm/v7"},
			},
			{
				ID:         "clean",
				Dockerfile: "./testdata/Dockerfile.clean",
				Images:     []string{"image3", "image4"},
				Tags:       []string{"tag3"},
				ExtraFiles: []string{"./testdata/foo.conf"},
				IDs:        []string{"nopenopenope"},
			},
		},
	}, testctx.Snapshot)
	for _, arch := range []string{"amd64", "arm64", "arm"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Path:   "./testdata/mybin",
			Goos:   "linux",
			Goarch: arch,
			Goarm:  "7",
			Type:   artifact.Binary,
			Extra: artifact.Extras{
				artifact.ExtraID: "id1",
			},
		})
	}

	require.NoError(t, Base{}.Default(ctx))
	err := Snapshot{}.Run(ctx)
	require.NoError(t, err, "message: %s, output: %v", gerrors.MessageOf(err), gerrors.DetailsOf(err))

	images := ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByType(artifact.DockerImageV2),
			artifact.ByIDs("myimg"),
		),
	).List()
	require.Len(t, images, 12)
	require.Equal(t, []string{
		"image1:tag1-amd64",
		"image1:tag1-arm64",
		"image1:tag1-armv7",
		"image1:tag2-amd64",
		"image1:tag2-arm64",
		"image1:tag2-armv7",
		"image2:tag1-amd64",
		"image2:tag1-arm64",
		"image2:tag1-armv7",
		"image2:tag2-amd64",
		"image2:tag2-arm64",
		"image2:tag2-armv7",
	}, names(images))
	for _, img := range images {
		require.NotEmpty(t, artifact.ExtraOr(*img, artifact.ExtraDigest, ""))
		rmi(t, img.Name)
	}

	images = ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByType(artifact.DockerImageV2),
			artifact.ByIDs("clean"),
		),
	).List()
	require.Equal(t, []string{
		"image3:tag3-amd64",
		"image3:tag3-arm64",
		"image4:tag3-amd64",
		"image4:tag3-arm64",
	}, names(images))
	for _, img := range images {
		require.NotEmpty(t, artifact.ExtraOr(*img, artifact.ExtraDigest, ""))
		rmi(t, img.Name)
	}
}

func TestPublish(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "registry images only available for windows")

	testlib.StartRegistry(t, "registry-v2", "5060")
	testlib.StartRegistry(t, "alt_registry-v2", "5061")

	dist := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			ProjectName: "dockerv2",
			Dist:        dist,
			DockersV2: []config.DockerV2{
				{
					ID:         "myimg",
					Dockerfile: "./testdata/Dockerfile",
					Images:     []string{"localhost:5060/foo", "localhost:5061/bar"},
					Tags:       []string{"latest", "v{{.Version}}", "{{if .IsNightly}}nightly{{end}}"},
					ExtraFiles: []string{"./testdata/foo.conf"},
					IDs:        []string{"id1"},
				},
			},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithCommit("a1b2c3d4"),
		testctx.WithSemver(1, 0, 0, ""),
	)
	for _, arch := range []string{"amd64", "arm64"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "mybin",
			Path:   "./testdata/mybin",
			Goos:   "linux",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: artifact.Extras{
				artifact.ExtraID: "id1",
			},
		})
	}

	require.NoError(t, Base{}.Default(ctx))
	err := Publish{}.Publish(ctx)
	require.NoError(t, err, "message: %s, output: %v", gerrors.MessageOf(err), gerrors.DetailsOf(err))

	images := ctx.Artifacts.Filter(artifact.ByType(artifact.DockerImageV2)).List()
	require.Len(t, images, 4)
	require.Equal(t, []string{
		"localhost:5060/foo:latest",
		"localhost:5060/foo:v1.0.0",
		"localhost:5061/bar:latest",
		"localhost:5061/bar:v1.0.0",
	}, names(images))

	for _, img := range images {
		require.NotEmpty(t, artifact.ExtraOr(*img, artifact.ExtraDigest, ""))
	}
}

func names(in []*artifact.Artifact) []string {
	out := make([]string, 0, len(in))
	for _, art := range in {
		out = append(out, art.Name)
	}
	slices.Sort(out)
	return out
}

func rmi(tb testing.TB, img string) {
	tb.Helper()
	require.NoError(tb, exec.CommandContext(tb.Context(), "docker", "rmi", "--force", img).Run())
}
