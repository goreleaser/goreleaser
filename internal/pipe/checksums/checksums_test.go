package checksums

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestPipe(t *testing.T) {
	const binary = "binary"
	const archive = binary + ".tar.gz"
	const linuxPackage = binary + ".rpm"
	const checksums = binary + "_bar_checksums.txt"
	const sum = "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  "

	tests := map[string]struct {
		ids  []string
		want string
	}{
		"default": {
			want: strings.Join([]string{
				sum + binary,
				sum + linuxPackage,
				sum + archive,
			}, "\n") + "\n",
		},
		"select ids": {
			ids: []string{
				"id-1",
				"id-2",
			},
			want: strings.Join([]string{
				sum + binary,
				sum + archive,
			}, "\n") + "\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			file := filepath.Join(folder, binary)
			require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: binary,
					Checksum: config.Checksum{
						NameTemplate: "{{ .ProjectName }}_{{ .Env.FOO }}_checksums.txt",
						Algorithm:    "sha256",
						IDs:          tt.ids,
					},
					Env: []string{"FOO=bar"},
				},
				testctx.WithCurrentTag("1.2.3"),
			)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: binary,
				Path: file,
				Type: artifact.UploadableBinary,
				Extra: map[string]any{
					artifact.ExtraID: "id-1",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: archive,
				Path: file,
				Type: artifact.UploadableArchive,
				Extra: map[string]any{
					artifact.ExtraID: "id-2",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: linuxPackage,
				Path: file,
				Type: artifact.LinuxPackage,
				Extra: map[string]any{
					artifact.ExtraID: "id-3",
				},
			})
			require.NoError(t, Pipe{}.Run(ctx))
			var artifacts []string
			for _, a := range ctx.Artifacts.List() {
				artifacts = append(artifacts, a.Name)
				require.NoError(t, a.Refresh(), "refresh should not fail and yield same results as nothing changed")
			}
			require.Contains(t, artifacts, checksums, binary)
			bts, err := os.ReadFile(filepath.Join(folder, checksums))
			require.NoError(t, err)
			require.Contains(t, tt.want, string(bts))
		})
	}
}

func TestPipeSplit(t *testing.T) {
	const binary = "binary"
	const archive = binary + ".tar.gz"
	const linuxPackage = binary + ".rpm"
	const sum = "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc"

	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: "foo",
			Checksum: config.Checksum{
				Split: true,
			},
		},
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
		Extra: map[string]any{
			artifact.ExtraID: "id-1",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: archive,
		Path: file,
		Type: artifact.UploadableArchive,
		Extra: map[string]any{
			artifact.ExtraID: "id-2",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: linuxPackage,
		Path: file,
		Type: artifact.LinuxPackage,
		Extra: map[string]any{
			artifact.ExtraID: "id-3",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))

	require.NoError(t, ctx.Artifacts.Visit(func(a *artifact.Artifact) error {
		return a.Refresh()
	}))

	checks := ctx.Artifacts.Filter(artifact.ByType(artifact.Checksum)).List()
	require.Len(t, checks, 3)

	for _, check := range checks {
		require.NotEmpty(t, check.Extra[artifact.ExtraChecksumOf])
		bts, err := os.ReadFile(check.Path)
		require.NoError(t, err)
		require.Equal(t, sum, string(bts))
	}
}

func TestRefreshModifying(t *testing.T) {
	const binary = "binary"
	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: binary,
		Checksum: config.Checksum{
			NameTemplate: "{{ .ProjectName }}_{{ .Env.FOO }}_checksums.txt",
			Algorithm:    "sha256",
		},
		Env: []string{"FOO=bar"},
	}, testctx.WithCurrentTag("1.2.3"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
	})
	require.NoError(t, Pipe{}.Run(ctx))
	checks := ctx.Artifacts.Filter(artifact.ByType(artifact.Checksum)).List()
	require.Len(t, checks, 1)
	previous, err := os.ReadFile(checks[0].Path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, []byte("some other string"), 0o644))
	require.NoError(t, checks[0].Refresh())
	current, err := os.ReadFile(checks[0].Path)
	require.NoError(t, err)
	require.NotEqual(t, string(previous), string(current))
}

func TestRefreshModifyingSplit(t *testing.T) {
	const binary = "binary"
	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
	ctx := testctx.NewWithCfg(config.Project{
		Dist:        folder,
		ProjectName: binary,
		Checksum: config.Checksum{
			Split: true,
		},
		Env: []string{"FOO=bar"},
	}, testctx.WithCurrentTag("1.2.3"))
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	checks := ctx.Artifacts.Filter(artifact.ByType(artifact.Checksum)).List()
	require.Len(t, checks, 1)
	previous, err := os.ReadFile(checks[0].Path)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(file, []byte("some other string"), 0o644))
	require.NoError(t, checks[0].Refresh())
	current, err := os.ReadFile(checks[0].Path)
	require.NoError(t, err)
	require.NotEqual(t, string(previous), string(current))
}

func TestPipeFileNotExist(t *testing.T) {
	folder := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
		testctx.WithCurrentTag("1.2.3"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "nope",
		Path: "/nope",
		Type: artifact.UploadableBinary,
	})
	require.ErrorIs(t, Pipe{}.Run(ctx), os.ErrNotExist)
}

func TestPipeInvalidNameTemplate(t *testing.T) {
	binFile, err := os.CreateTemp(t.TempDir(), "goreleasertest-bin")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, binFile.Close()) })
	_, err = binFile.WriteString("fake artifact")
	require.NoError(t, err)

	for _, template := range []string{
		"{{ .Pro }_checksums.txt",
		"{{.Env.NOPE}}",
	} {
		for _, split := range []bool{true, false} {
			t.Run(fmt.Sprintf("split_%v_%s", split, template), func(t *testing.T) {
				folder := t.TempDir()
				ctx := testctx.NewWithCfg(
					config.Project{
						Dist:        folder,
						ProjectName: "name",
						Checksum: config.Checksum{
							NameTemplate: template,
							Algorithm:    "sha256",
							Split:        split,
						},
					},
					testctx.WithCurrentTag("1.2.3"),
				)
				ctx.Artifacts.Add(&artifact.Artifact{
					Name: "whatever",
					Type: artifact.UploadableBinary,
					Path: binFile.Name(),
				})
				testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
			})
		}
	}
}

func TestPipeWhenNoArtifacts(t *testing.T) {
	ctx := testctx.New()
	require.NoError(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.Artifacts.List())
}

func TestDefault(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Checksum: config.Checksum{},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(
		t,
		"{{ .ProjectName }}_{{ .Version }}_checksums.txt",
		ctx.Config.Checksum.NameTemplate,
	)
	require.Equal(t, "sha256", ctx.Config.Checksum.Algorithm)
}

func TestDefaultSPlit(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Checksum: config.Checksum{
			Split: true,
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(
		t,
		"{{ .ArtifactName }}.{{ .Algorithm }}",
		ctx.Config.Checksum.NameTemplate,
	)
	require.Equal(t, "sha256", ctx.Config.Checksum.Algorithm)
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Checksum: config.Checksum{
			NameTemplate: "checksums.txt",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "checksums.txt", ctx.Config.Checksum.NameTemplate)
}

func TestPipeChecksumsSortByFilename(t *testing.T) {
	const binary = "binary"
	const checksums = "checksums.txt"
	const filePaths = "./testdata/order/*.txt"

	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))

	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: binary,
			Checksum: config.Checksum{
				Algorithm:    "sha256",
				NameTemplate: "checksums.txt",
				ExtraFiles: []config.ExtraFile{
					{Glob: filePaths},
				},
			},
		},
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Run(ctx))

	bts, err := os.ReadFile(filepath.Join(folder, checksums))
	require.NoError(t, err)

	lines := strings.Split(string(bts), "\n")

	wantLinesOrder := []string{
		"ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb  a.txt",
		"3b64db95cb55c763391c707108489ae18b4112d783300de38e033b4c98c3deaf  b.txt",
		"61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  binary",
		"64daa44ad493ff28a96effab6e77f1732a3d97d83241581b37dbd70a7a4900fe  c.txt",
		"5bf8aa57fc5a6bc547decf1cc6db63f10deb55a3c6c5df497d631fb3d95e1abf  d.txt",
	}

	for i, want := range wantLinesOrder {
		require.Equal(t, want, lines[i])
	}
}

func TestPipeCheckSumsWithExtraFiles(t *testing.T) {
	const binary = "binary"
	const checksums = "checksums.txt"
	const extraFileFooRelPath = "./testdata/foo.txt"
	const extraFileBarRelPath = "./testdata/**/bar.txt"
	const extraFileFoo = "foo.txt"
	const extraFileBar = "bar.txt"

	tests := map[string]struct {
		extraFiles []config.ExtraFile
		ids        []string
		want       []string
	}{
		"default": {
			extraFiles: nil,
			want: []string{
				binary,
			},
		},
		"one extra file": {
			extraFiles: []config.ExtraFile{
				{Glob: extraFileFooRelPath},
			},
			want: []string{
				extraFileFoo,
			},
		},
		"multiple extra files": {
			extraFiles: []config.ExtraFile{
				{Glob: extraFileFooRelPath},
				{Glob: extraFileBarRelPath},
			},
			want: []string{
				extraFileFoo,
				extraFileBar,
			},
		},
		"one extra file with no builds": {
			extraFiles: []config.ExtraFile{
				{Glob: extraFileFooRelPath},
			},
			ids: []string{"yada yada yada"},
			want: []string{
				extraFileFoo,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			file := filepath.Join(folder, binary)
			require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: binary,
					Checksum: config.Checksum{
						Algorithm:    "sha256",
						NameTemplate: "checksums.txt",
						ExtraFiles:   tt.extraFiles,
						IDs:          tt.ids,
					},
				},
			)

			ctx.Artifacts.Add(&artifact.Artifact{
				Name: binary,
				Path: file,
				Type: artifact.UploadableBinary,
				Extra: map[string]any{
					artifact.ExtraID: "id-1",
				},
			})

			require.NoError(t, Pipe{}.Run(ctx))

			bts, err := os.ReadFile(filepath.Join(folder, checksums))

			require.NoError(t, err)
			for _, want := range tt.want {
				if tt.extraFiles == nil {
					require.Contains(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  "+want)
				} else {
					require.Contains(t, string(bts), "3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  "+want)
				}
			}

			_ = ctx.Artifacts.Visit(func(a *artifact.Artifact) error {
				if a.Path != file {
					return nil
				}
				if len(tt.ids) > 0 {
					return nil
				}
				checkSum := artifact.MustExtra[string](*a, artifactChecksumExtra)
				require.NotEmptyf(t, checkSum, "failed: %v", a.Path)
				return nil
			})
		})
	}
}

func TestExtraFilesNoMatch(t *testing.T) {
	dir := t.TempDir()
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        dir,
			ProjectName: "fake",
			Checksum: config.Checksum{
				Algorithm:    "sha256",
				NameTemplate: "checksums.txt",
				ExtraFiles:   []config.ExtraFile{{Glob: "./nope/nope.txt"}},
			},
		},
	)

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "fake",
		Path: "fake-path",
		Type: artifact.UploadableBinary,
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.EqualError(t, Pipe{}.Run(ctx), `globbing failed for pattern ./nope/nope.txt: matching "./nope/nope.txt": file does not exist`)
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			Checksum: config.Checksum{
				Disable: true,
			},
		})
		require.True(t, Pipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		require.False(t, Pipe{}.Skip(testctx.New()))
	})
}

// TODO: add tests for LinuxPackage and UploadableSourceArchive
