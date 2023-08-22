package archive

import (
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestMeta(t *testing.T) {
	t.Run("good", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Meta:         true,
					NameTemplate: "foo",
					Files: []config.File{
						{Source: "testdata/**/*.txt"},
					},
				},
			},
		})

		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		require.Equal(
			t,
			[]string{"testdata/a/a.txt", "testdata/a/b/a.txt", "testdata/a/b/c/d.txt"},
			testlib.LsArchive(t, filepath.Join(dist, "foo.tar.gz"), "tar.gz"),
		)
	})

	t.Run("bad tmpl", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Meta:         true,
					NameTemplate: "foo{{.Os}}",
					Files: []config.File{
						{Source: "testdata/**/*.txt"},
					},
				},
			},
		})

		require.NoError(t, Pipe{}.Default(ctx))
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("no files", func(t *testing.T) {
		dist := t.TempDir()
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			Archives: []config.Archive{
				{
					Meta:         true,
					NameTemplate: "foo",
				},
			},
		})

		require.NoError(t, Pipe{}.Default(ctx))
		require.EqualError(t, Pipe{}.Run(ctx), `no files found`)
	})
}
