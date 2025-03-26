package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/golden"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunWithError(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        "testadata/nope",
		ProjectName: "foo",
	})
	require.ErrorIs(t, MetaPipe{}.Run(ctx), os.ErrNotExist)
	require.ErrorIs(t, ArtifactsPipe{}.Run(ctx), os.ErrNotExist)
}

func TestRun(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(time.Second).UTC()

	getCtx := func(tmp string) *context.Context {
		ctx := testctx.NewWithCfg(
			config.Project{
				Dist:        tmp,
				ProjectName: "name",
				Metadata: config.ProjectMetadata{
					ModTimestamp: "{{.Env.MOD_TS}}",
				},
			},
			testctx.WithPreviousTag("v1.2.2"),
			testctx.WithCurrentTag("v1.2.3"),
			testctx.WithCommit("aef34a"),
			testctx.WithVersion("1.2.3"),
			testctx.WithDate(time.Date(2022, 0o1, 22, 10, 12, 13, 0, time.UTC)),
			testctx.WithFakeRuntime,
			testctx.WithEnv(map[string]string{
				"MOD_TS": fmt.Sprintf("%d", modTime.Unix()),
			}),
		)
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:   "foo",
			Path:   "foo.txt",
			Type:   artifact.Binary,
			Goos:   "darwin",
			Goarch: "amd64",
			Goarm:  "7",
			Extra: map[string]any{
				"foo": "bar",
			},
		})
		return ctx
	}

	t.Run("artifacts", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, Pipe{}.Run(ctx))
		require.NoError(t, ArtifactsPipe{}.Run(ctx))
		requireEqualJSONFile(t, filepath.Join(tmp, "artifacts.json"), modTime)
	})

	t.Run("metadata", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, Pipe{}.Run(ctx))
		require.NoError(t, MetaPipe{}.Run(ctx))

		metas := ctx.Artifacts.Filter(artifact.ByType(artifact.Metadata)).List()
		require.Len(t, metas, 1)
		require.Equal(t, "metadata.json", metas[0].Name)
		requireEqualJSONFile(t, metas[0].Path, modTime)
	})

	t.Run("invalid mod metadata", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		ctx.Config.Metadata.ModTimestamp = "not a number"
		require.NoError(t, Pipe{}.Run(ctx))
		require.ErrorIs(t, MetaPipe{}.Run(ctx), strconv.ErrSyntax)
		require.ErrorIs(t, ArtifactsPipe{}.Run(ctx), strconv.ErrSyntax)
	})

	t.Run("invalid mod metadata tmpl", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		ctx.Config.Metadata.ModTimestamp = "{{.Nope}}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func requireEqualJSONFile(tb testing.TB, path string, modTime time.Time) {
	tb.Helper()
	golden.RequireEqualJSON(tb, golden.RequireReadFile(tb, path))
	stat, err := os.Stat(path)
	require.NoError(tb, err)
	require.Equal(tb, modTime.Unix(), stat.ModTime().Unix())
}
