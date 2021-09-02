package checksums

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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

	tests := map[string]struct {
		ids  []string
		want []string
	}{
		"default": {
			want: []string{
				binary,
				archive,
				linuxPackage,
			},
		},
		"select ids": {
			ids: []string{
				"id-1",
				"id-2",
			},
			want: []string{
				binary,
				archive,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			file := filepath.Join(folder, binary)
			require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
			ctx := context.New(
				config.Project{
					Dist:        folder,
					ProjectName: binary,
					Checksum: config.Checksum{
						NameTemplate: "{{ .ProjectName }}_{{ .Env.FOO }}_checksums.txt",
						Algorithm:    "sha256",
						IDs:          tt.ids,
					},
				},
			)
			ctx.Git.CurrentTag = "1.2.3"
			ctx.Env = map[string]string{"FOO": "bar"}
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: binary,
				Path: file,
				Type: artifact.UploadableBinary,
				Extra: map[string]interface{}{
					"ID": "id-1",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: archive,
				Path: file,
				Type: artifact.UploadableArchive,
				Extra: map[string]interface{}{
					"ID": "id-2",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: linuxPackage,
				Path: file,
				Type: artifact.LinuxPackage,
				Extra: map[string]interface{}{
					"ID": "id-3",
				},
			})
			require.NoError(t, Pipe{}.Run(ctx))
			var artifacts []string
			for _, a := range ctx.Artifacts.List() {
				artifacts = append(artifacts, a.Name)
			}
			require.Contains(t, artifacts, checksums, binary)
			bts, err := os.ReadFile(filepath.Join(folder, checksums))
			require.NoError(t, err)
			for _, want := range tt.want {
				require.Contains(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  "+want)
			}
		})
	}
}

func TestPipeSkipTrue(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				Disable: true,
			},
		},
	)
	err := Pipe{}.Run(ctx)
	testlib.AssertSkipped(t, err)
	require.EqualError(t, err, pipe.ErrSkipDisabledPipe.Error())
}

func TestPipeFileNotExist(t *testing.T) {
	folder := t.TempDir()
	ctx := context.New(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	)
	ctx.Git.CurrentTag = "1.2.3"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "nope",
		Path: "/nope",
		Type: artifact.UploadableBinary,
	})
	err := Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "/nope: no such file or directory")
}

func TestPipeInvalidNameTemplate(t *testing.T) {
	binFile, err := ioutil.TempFile(t.TempDir(), "goreleasertest-bin")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, binFile.Close()) })
	_, err = binFile.WriteString("fake artifact")
	require.NoError(t, err)

	for template, eerr := range map[string]string{
		"{{ .Pro }_checksums.txt": `template: tmpl:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}":           `template: tmpl:1:6: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(t *testing.T) {
			folder := t.TempDir()
			ctx := context.New(
				config.Project{
					Dist:        folder,
					ProjectName: "name",
					Checksum: config.Checksum{
						NameTemplate: template,
						Algorithm:    "sha256",
					},
				},
			)
			ctx.Git.CurrentTag = "1.2.3"
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: "whatever",
				Type: artifact.UploadableBinary,
				Path: binFile.Name(),
			})
			err = Pipe{}.Run(ctx)
			require.Error(t, err)
			require.Equal(t, eerr, err.Error())
		})
	}
}

func TestPipeCouldNotOpenChecksumsTxt(t *testing.T) {
	folder := t.TempDir()
	binFile, err := ioutil.TempFile(folder, "goreleasertest-bin")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, binFile.Close()) })
	_, err = binFile.WriteString("fake artifact")
	require.NoError(t, err)

	file := filepath.Join(folder, "checksums.txt")
	require.NoError(t, os.WriteFile(file, []byte("some string"), 0o000))
	ctx := context.New(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
				Algorithm:    "sha256",
			},
		},
	)
	ctx.Git.CurrentTag = "1.2.3"
	ctx.Artifacts.Add(&artifact.Artifact{
		Name: "whatever",
		Type: artifact.UploadableBinary,
		Path: binFile.Name(),
	})
	err = Pipe{}.Run(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "/checksums.txt: permission denied")
}

func TestPipeWhenNoArtifacts(t *testing.T) {
	ctx := &context.Context{}
	require.NoError(t, Pipe{}.Run(ctx))
	require.Len(t, ctx.Artifacts.List(), 0)
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Checksum: config.Checksum{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(
		t,
		"{{ .ProjectName }}_{{ .Version }}_checksums.txt",
		ctx.Config.Checksum.NameTemplate,
	)
	require.Equal(t, "sha256", ctx.Config.Checksum.Algorithm)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "checksums.txt", ctx.Config.Checksum.NameTemplate)
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
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			file := filepath.Join(folder, binary)
			require.NoError(t, os.WriteFile(file, []byte("some string"), 0o644))
			ctx := context.New(
				config.Project{
					Dist:        folder,
					ProjectName: binary,
					Checksum: config.Checksum{
						Algorithm:    "sha256",
						NameTemplate: "checksums.txt",
						ExtraFiles:   tt.extraFiles,
					},
				},
			)

			ctx.Artifacts.Add(&artifact.Artifact{
				Name: binary,
				Path: file,
				Type: artifact.UploadableBinary,
				Extra: map[string]interface{}{
					"ID": "id-1",
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
		})
	}
}

// TODO: add tests for LinuxPackage and UploadableSourceArchive
