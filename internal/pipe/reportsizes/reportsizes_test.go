package reportsizes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})
	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.NewWithCfg(config.Project{
			ReportSizes: true,
		})))
	})
}

func TestRun(t *testing.T) {
	ctx := testctx.New()
	for i, tp := range []artifact.Type{
		artifact.Binary,
		artifact.UniversalBinary,
		artifact.UploadableArchive,
		artifact.PublishableSnapcraft,
		artifact.LinuxPackage,
		artifact.CArchive,
		artifact.CShared,
		artifact.Header,
	} {
		if i%2 == 0 {
			cw, err := os.Getwd()
			require.NoError(t, err)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:  "foo",
				Path:  filepath.Join(cw, "reportsizes.go"),
				Extra: map[string]any{},
				Type:  tp,
			})
			continue
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name:  "foo",
			Path:  "reportsizes.go",
			Extra: map[string]any{},
			Type:  tp,
		})
	}

	require.NoError(t, Pipe{}.Run(ctx))

	for _, art := range ctx.Artifacts.List() {
		require.NotZero(t, artifact.ExtraOr[int64](*art, artifact.ExtraSize, 0))
	}
}
