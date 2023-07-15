package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunWithError(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        "testadata/nope",
		ProjectName: "foo",
	})
	require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
}

func TestRun(t *testing.T) {
	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()

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
			Extra: map[string]interface{}{
				"foo": "bar",
			},
		})
		return ctx
	}

	t.Run("artifacts", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, Pipe{}.Run(ctx))
		requireEqualJSONFile(t, tmp, "artifacts.json", modTime)
	})
	t.Run("metadata", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		require.NoError(t, Pipe{}.Run(ctx))
		requireEqualJSONFile(t, tmp, "metadata.json", modTime)
	})

	t.Run("invalid mod metadata", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		ctx.Config.Metadata.ModTimestamp = "not a number"
		require.ErrorIs(t, Pipe{}.Run(ctx), strconv.ErrSyntax)
	})

	t.Run("invalid mod metadata tmpl", func(t *testing.T) {
		tmp := t.TempDir()
		ctx := getCtx(tmp)
		ctx.Config.Metadata.ModTimestamp = "{{.Nope}}"
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})
}

func requireEqualJSONFile(tb testing.TB, tmp, s string, modTime time.Time) {
	tb.Helper()
	path := filepath.Join(tmp, s)
	golden.RequireEqualJSON(tb, golden.RequireReadFile(tb, path))
	stat, err := os.Stat(path)
	require.NoError(tb, err)
	require.Equal(tb, modTime.Unix(), stat.ModTime().Unix())
}
