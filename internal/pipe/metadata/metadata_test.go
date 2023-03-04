package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
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
	tmp := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        tmp,
			ProjectName: "name",
		},
		testctx.WithPreviousTag("v1.2.2"),
		testctx.WithCurrentTag("v1.2.3"),
		testctx.WithCommit("aef34a"),
		testctx.WithVersion("1.2.3"),
		testctx.WithDate(time.Date(2022, 0o1, 22, 10, 12, 13, 0, time.UTC)),
		testctx.WithFakeRuntime,
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

	require.NoError(t, Pipe{}.Run(ctx))
	t.Run("artifacts", func(t *testing.T) {
		requireEqualJSONFile(t, tmp, "artifacts.json")
	})
	t.Run("metadata", func(t *testing.T) {
		requireEqualJSONFile(t, tmp, "metadata.json")
	})
}

func requireEqualJSONFile(tb testing.TB, tmp, s string) {
	tb.Helper()
	path := filepath.Join(tmp, s)
	golden.RequireEqualJSON(tb, golden.RequireReadFile(tb, path))

	info, err := os.Stat(path)
	require.NoError(tb, err)
	require.Equal(tb, "-rw-r--r--", info.Mode().String())
}
