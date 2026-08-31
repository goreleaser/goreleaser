package chocolatey

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

// TestRunSkippedChocolateyDoesNotStopOthers covers the same shape as
// winget's TestRunPipeSkippedWingetDoesNotStopOthers: a chocolatey entry
// whose config errors on multiple archives for the same platform must not
// stop the valid entries after it.
func TestRunSkippedChocolateyDoesNotStopOthers(t *testing.T) {
	folder := t.TempDir()
	file := filepath.Join(folder, "archive")
	require.NoError(t, os.WriteFile(file, []byte("lorem ipsum"), 0o644))

	cmd = fakeCmd{execFn: func(_ string, _ ...string) ([]byte, error) {
		return []byte("success"), nil
	}}
	t.Cleanup(func() { cmd = stdCmd{} })

	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist:        folder,
		ProjectName: "foo",
		Chocolateys: []config.Chocolatey{
			// two amd64 archives for the same entry: skipped by
			// errMultipleArchives.
			{Name: "skipped", IDs: []string{"dupe"}, Goamd64: "v1"},
			// valid entry pointing at the 386 archive.
			{Name: "valid", IDs: []string{"good"}, Goamd64: "v1"},
		},
	}, testctx.WithCurrentTag("v1.0.1"), testctx.WithVersion("1.0.1"), testctx.GitHubTokenType)

	for _, n := range []string{"a", "b"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:    n + "_1.0.1_windows_amd64.zip",
			Path:    file,
			Goos:    "windows",
			Goarch:  "amd64",
			Goamd64: "v1",
			Type:    artifact.UploadableArchive,
			Extra: map[string]any{
				artifact.ExtraID:     "dupe",
				artifact.ExtraFormat: "zip",
			},
		})
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:    "good_1.0.1_windows_386.zip",
		Path:    file,
		Goos:    "windows",
		Goarch:  "386",
		Goamd64: "v1",
		Type:    artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID:     "good",
			artifact.ExtraFormat: "zip",
		},
	})

	p := Pipe{}
	require.NoError(t, p.Default(ctx))

	err := p.Run(ctx)
	require.True(t, pipe.IsSkip(err), "expected a skip error, got %v", err)

	// the entry after the skipped one must still have produced its package.
	require.Len(t, ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableChocolatey)).List(), 1)
}
