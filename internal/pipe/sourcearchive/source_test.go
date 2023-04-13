package sourcearchive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	for _, format := range []string{"tar.gz", "tar", "zip"} {
		t.Run(format, func(t *testing.T) {
			tmp := testlib.Mktmp(t)
			require.NoError(t, os.Mkdir("dist", 0o744))

			testlib.GitInit(t)
			require.NoError(t, os.WriteFile("code.rb", []byte("not really code"), 0o655))
			require.NoError(t, os.WriteFile("code.py", []byte("print 1"), 0o655))
			require.NoError(t, os.WriteFile("README.md", []byte("# my dope fake project"), 0o655))
			require.NoError(t, os.WriteFile(".gitignore", []byte(`
added-later.txt
ignored.txt
code.txt
subfolder/
			`), 0o655))

			require.NoError(t, os.WriteFile(".gitattributes", []byte(`
.VERSION export-subst
			`), 0o655))
			require.NoError(t, os.WriteFile(".VERSION", []byte("$Format:%d$"), 0o655))
			testlib.GitAdd(t)
			testlib.GitCommit(t, "feat: first")
			testlib.GitTag(t, "v1.0.0")
			require.NoError(t, os.WriteFile("added-later.txt", []byte("this file was added later"), 0o655))
			require.NoError(t, os.WriteFile("ignored.md", []byte("never added"), 0o655))
			require.NoError(t, os.WriteFile("code.txt", []byte("not really code"), 0o655))
			require.NoError(t, os.MkdirAll("subfolder", 0o755))
			require.NoError(t, os.WriteFile("subfolder/file.md", []byte("a file within a folder, added later"), 0o655))

			t.Run("with extra files", func(t *testing.T) {
				ctx := testctx.NewWithCfg(
					config.Project{
						ProjectName: "foo",
						Dist:        "dist",
						Source: config.Source{
							Format:         format,
							Enabled:        true,
							PrefixTemplate: "{{ .ProjectName }}-{{ .Version }}/",
							Files: []config.File{
								{Source: "*.txt"},
								{Source: "subfolder/*"},
							},
						},
					},
					testctx.WithCommit("HEAD"),
					testctx.WithVersion("1.0.0"),
					testctx.WithCurrentTag("v1.0.0"),
				)

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

				expected := []string{
					"foo-1.0.0/",
					"foo-1.0.0/.gitignore",
					"foo-1.0.0/.gitattributes",
					"foo-1.0.0/.VERSION",
					"foo-1.0.0/README.md",
					"foo-1.0.0/code.py",
					"foo-1.0.0/code.rb",
					"foo-1.0.0/code.txt",
					"foo-1.0.0/added-later.txt",
					"foo-1.0.0/subfolder/file.md",
				}

				// zips wont have the parent dir
				if format == "zip" {
					expected = expected[1:]
				}

				require.ElementsMatch(t, expected, testlib.LsArchive(t, path, format))

				version := testlib.GetFileFromArchive(t, path, format, "foo-1.0.0/.VERSION")
				require.Equal(t, " (HEAD -> main, tag: v1.0.0)", string(version))
			})

			t.Run("simple", func(t *testing.T) {
				ctx := testctx.NewWithCfg(
					config.Project{
						ProjectName: "foo",
						Dist:        "dist",
						Source: config.Source{
							Format:         format,
							Enabled:        true,
							PrefixTemplate: "{{ .ProjectName }}-{{ .Version }}/",
						},
					},
					testctx.WithCommit("HEAD"),
					testctx.WithVersion("1.0.0"),
					testctx.WithCurrentTag("v1.0.0"),
				)

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

				expected := []string{
					"foo-1.0.0/",
					"foo-1.0.0/.gitignore",
					"foo-1.0.0/.gitattributes",
					"foo-1.0.0/.VERSION",
					"foo-1.0.0/README.md",
					"foo-1.0.0/code.py",
					"foo-1.0.0/code.rb",
				}

				// zips wont have the parent dir
				if format == "zip" {
					expected = expected[1:]
				}

				require.ElementsMatch(t, expected, testlib.LsArchive(t, path, format))

				version := testlib.GetFileFromArchive(t, path, format, "foo-1.0.0/.VERSION")
				require.Equal(t, " (HEAD -> main, tag: v1.0.0)", string(version))
			})
		})
	}
}

func TestInvalidFormat(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        t.TempDir(),
		ProjectName: "foo",
		Source: config.Source{
			Format:         "7z",
			Enabled:        true,
			PrefixTemplate: "{{ .ProjectName }}-{{ .Version }}/",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), "invalid source archive format: 7z")
}

func TestDefault(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, config.Source{
		NameTemplate: "{{ .ProjectName }}-{{ .Version }}",
		Format:       "tar.gz",
	}, ctx.Config.Source)
}

func TestInvalidNameTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Source: config.Source{
			Enabled:      true,
			NameTemplate: "{{ .foo }-asdda",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestInvalidInvalidFileTemplate(t *testing.T) {
	testlib.Mktmp(t)
	require.NoError(t, os.Mkdir("dist", 0o744))

	testlib.GitInit(t)
	require.NoError(t, os.WriteFile("code.txt", []byte("not really code"), 0o655))
	testlib.GitAdd(t)
	testlib.GitCommit(t, "feat: first")

	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
		Dist:        "dist",
		Source: config.Source{
			Format:  "tar.gz",
			Enabled: true,
			Files: []config.File{
				{Source: "{{.Test}"},
			},
		},
	})
	ctx.Git.FullCommit = "HEAD"
	ctx.Version = "1.0.0"
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestInvalidPrefixTemplate(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Dist: t.TempDir(),
		Source: config.Source{
			Enabled:        true,
			PrefixTemplate: "{{ .ProjectName }/",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestDisabled(t *testing.T) {
	require.True(t, Pipe{}.Skip(testctx.New()))
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Source: config.Source{
				Enabled: true,
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestString(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}
