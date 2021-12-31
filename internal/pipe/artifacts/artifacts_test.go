package artifacts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/golden"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestArtifacts(t *testing.T) {
	tmp := t.TempDir()
	ctx := context.New(config.Project{
		Dist: tmp,
	})

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
	path := filepath.Join(tmp, "artifacts.json")
	golden.RequireEqualJSON(t, golden.RequireReadFile(t, path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, "-rw-r--r--", info.Mode().String())
}
