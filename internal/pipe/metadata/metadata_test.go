package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestRunWithError(t *testing.T) {
	ctx := context.New(config.Project{
		Dist:        "testadata/nope",
		ProjectName: "foo",
	})
	require.EqualError(t, Pipe{}.Run(ctx), `open testadata/nope/artifacts.json: no such file or directory`)
}

func TestRun(t *testing.T) {
	tmp := t.TempDir()
	ctx := context.New(config.Project{
		Dist:        tmp,
		ProjectName: "name",
	})
	ctx.Version = "1.2.3"
	ctx.Git = context.GitInfo{
		CurrentTag:  "v1.2.3",
		PreviousTag: "v1.2.2",
		Commit:      "aef34a",
	}
	ctx.Date = time.Date(2022, 0o1, 22, 10, 12, 13, 0, time.UTC)
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
