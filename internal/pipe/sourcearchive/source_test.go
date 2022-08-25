package sourcearchive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	for _, format := range []string{"tar.gz", "tar", "zip"} {
		t.Run(format, func(t *testing.T) {
			tmp := testlib.Mktmp(t)
			require.NoError(t, os.Mkdir("dist", 0o744))

			testlib.GitInit(t)
			require.NoError(t, os.WriteFile("code.txt", []byte("not really code"), 0o655))
			require.NoError(t, os.WriteFile("code.py", []byte("print 1"), 0o655))
			require.NoError(t, os.WriteFile("README.md", []byte("# my dope fake project"), 0o655))
			testlib.GitAdd(t)
			testlib.GitCommit(t, "feat: first")
			require.NoError(t, os.WriteFile("added-later.txt", []byte("this file was added later"), 0o655))
			require.NoError(t, os.WriteFile("ignored.md", []byte("never added"), 0o655))

			ctx := context.New(config.Project{
				ProjectName: "foo",
				Dist:        "dist",
				Source: config.Source{
					Format:         format,
					Enabled:        true,
					PrefixTemplate: "{{ .ProjectName }}-{{ .Version }}/",
					Files: []string{
						"*.txt",
					},
				},
			})
			ctx.Git.FullCommit = "HEAD"
			ctx.Version = "1.0.0"

			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))

			artifacts := ctx.Artifacts.List()
			require.Len(t, artifacts, 1)
			require.Equal(t, artifact.Artifact{
				Type: artifact.UploadableSourceArchive,
				Name: "foo-1.0.0." + format,
				Path: "dist/foo-1.0.0." + format,
				Extra: map[string]interface{}{
					artifact.ExtraFormat: format,
				},
			}, *artifacts[0])
			path := filepath.Join(tmp, "dist", "foo-1.0.0."+format)
			stat, err := os.Stat(path)
			require.NoError(t, err)
			require.Greater(t, stat.Size(), int64(100))

			if format != "zip" {
				return
			}

			f, err := os.Open(path)
			require.NoError(t, err)
			z, err := zip.NewReader(f, stat.Size())
			require.NoError(t, err)

			var paths []string
			for _, zf := range z.File {
				paths = append(paths, zf.Name)
			}
			require.Equal(t, []string{
				"foo-1.0.0/",
				"foo-1.0.0/README.md",
				"foo-1.0.0/code.py",
				"foo-1.0.0/code.txt",
				"foo-1.0.0/added-later.txt",
			}, paths)
		})
	}
}

func TestInvalidFormat(t *testing.T) {
	ctx := context.New(config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		Source: config.Source{
			Format:         "7z",
			Enabled:        true,
			PrefixTemplate: "{{ .ProjectName }}-{{ .Version }}/",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "invalid archive format: 7z")
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, config.Source{
		NameTemplate: "{{ .ProjectName }}-{{ .Version }}",
		Format:       "tar.gz",
	}, ctx.Config.Source)
}

func TestInvalidNameTemplate(t *testing.T) {
	ctx := context.New(config.Project{
		Source: config.Source{
			Enabled:      true,
			NameTemplate: "{{ .foo }-asdda",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), "template: tmpl:1: unexpected \"}\" in operand")
}

func TestDisabled(t *testing.T) {
	require.True(t, Pipe{}.Skip(context.New(config.Project{})))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Source: config.Source{
				Enabled: true,
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}
