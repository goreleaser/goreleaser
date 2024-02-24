package checksums

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
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

	want := strings.Join([]string{
		sum + binary,
		sum + linuxPackage,
		sum + archive,
	}, "\n") + "\n"

	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: binary,
			Checksum: config.Checksum{
				NameTemplate: "{{ .ProjectName }}_{{ .Env.FOO }}_checksums.txt",
			},
			Env: []string{"FOO=bar"},
		},
		testctx.WithCurrentTag("1.2.3"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-1",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: archive,
		Path: file,
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-2",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: linuxPackage,
		Path: file,
		Type: artifact.LinuxPackage,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-3",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	var artifacts []string
	result, err := ctx.Artifacts.Checksums().List()
	require.NoError(t, err)
	for _, a := range result {
		artifacts = append(artifacts, a.Name)
	}
	require.Contains(t, artifacts, checksums, binary)
	bts, err := os.ReadFile(filepath.Join(folder, checksums))
	require.NoError(t, err)
	require.Contains(t, want, string(bts))
}

func TestPipeSplit(t *testing.T) {
	const binary = "binary"
	const archive = binary + ".tar.gz"
	const linuxPackage = binary + ".rpm"

	folder := t.TempDir()
	file := filepath.Join(folder, binary)
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist:        folder,
			ProjectName: binary,
			Checksum: config.Checksum{
				Split: true,
			},
		},
		testctx.WithCurrentTag("1.2.3"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-1",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: archive,
		Path: file,
		Type: artifact.UploadableArchive,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-2",
		},
	})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: linuxPackage,
		Path: file,
		Type: artifact.LinuxPackage,
		Extra: map[string]interface{}{
			artifact.ExtraID: "id-3",
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	result, err := ctx.Artifacts.Checksums().List()
	require.NoError(t, err)
	require.Len(t, result, 6)

	checks, err := ctx.Artifacts.Checksums().Get()
	require.Len(t, checks, 3)
	require.NoError(t, err)

	expected := map[string]string{
		"binary.rpm.sha256":    "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc",
		"binary.sha256":        "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc",
		"binary.tar.gz.sha256": "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc",
	}

	for _, check := range checks {
		sha, err := os.ReadFile(check.Path)
		require.NoError(t, err)

		got, ok := expected[check.Name]
		require.True(t, ok)
		require.Equal(t, got, string(sha))
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

	checks, err := ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary)).Checksums().List()
	require.NoError(t, err)
	require.Len(t, checks, 2)

	previous, err := os.ReadFile(checks[1].Path)
	require.NoError(t, err)
	require.Equal(t, "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  binary", string(previous))

	require.NoError(t, os.WriteFile(file, []byte("some other string"), 0o644))

	checks, err = ctx.Artifacts.Filter(artifact.ByType(artifact.UploadableBinary)).Checksums().List()
	require.NoError(t, err)
	require.Len(t, checks, 2)

	current, err := os.ReadFile(checks[1].Path)
	require.NoError(t, err)
	require.Equal(t, "94870326db59631f737f8392e49d18608d69018b1da3a79517a25623cd959c4c  binary", string(current))
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

	require.NoError(t, Pipe{}.Run(ctx))
	_, err := ctx.Artifacts.Checksums().List()
	require.ErrorIs(t, err, os.ErrNotExist)
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
		t.Run(template, func(t *testing.T) {
			folder := t.TempDir()
			ctx := testctx.NewWithCfg(
				config.Project{
					Dist:        folder,
					ProjectName: "name",
					Checksum: config.Checksum{
						NameTemplate: template,
						Algorithm:    "sha256",
					},
				},
				testctx.WithCurrentTag("1.2.3"),
			)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: "whatever",
				Type: artifact.UploadableBinary,
				Path: binFile.Name(),
			})
			require.NoError(t, Pipe{}.Run(ctx))
			_, err := ctx.Artifacts.Checksums().List()
			testlib.RequireTemplateError(t, err)
		})
	}
}

func TestPipeCouldNotOpenChecksumsTxt(t *testing.T) {
	folder := t.TempDir()
	binFile, err := os.CreateTemp(folder, "goreleasertest-bin")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, binFile.Close()) })
	_, err = binFile.WriteString("fake artifact")
	require.NoError(t, err)

	file := filepath.Join(folder, "checksums.txt")
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o000))
	ctx := testctx.NewWithCfg(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
				Algorithm:    "sha256",
			},
		},
		testctx.WithCurrentTag("1.2.3"),
	)
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "whatever",
		Type: artifact.UploadableBinary,
		Path: binFile.Name(),
	})
	require.NoError(t, Pipe{}.Run(ctx))
	_, err = ctx.Artifacts.Checksums().List()
	require.ErrorIs(t, err, syscall.EACCES)
}

func TestPipeWhenNoArtifacts(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		ProjectName: "foo",
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.NoError(t, Pipe{}.Run(ctx))
	list, err := ctx.Artifacts.Checksums().List()
	require.NoError(t, err)
	require.Empty(t, list)
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

func TestDefaultSplit(t *testing.T) {
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

	for _, f := range []string{"a", "b", "c", "d"} {
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: f + ".txt",
			Path: filepath.Join("testdata", "order", f+".txt"),
			Type: artifact.UploadableFile,
		})
	}

	require.NoError(t, Pipe{}.Run(ctx))
	_, err := ctx.Artifacts.Checksums().List()
	require.NoError(t, err)

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
