package checksums

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"

	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
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
			folder, err := ioutil.TempDir("", "goreleasertest")
			assert.NoError(t, err)
			var file = filepath.Join(folder, binary)
			assert.NoError(t, ioutil.WriteFile(file, []byte("some string"), 0644))
			var ctx = context.New(
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
			assert.NoError(t, Pipe{}.Run(ctx))
			var artifacts []string
			for _, a := range ctx.Artifacts.List() {
				artifacts = append(artifacts, a.Name)
			}
			assert.Contains(t, artifacts, checksums, binary)
			bts, err := ioutil.ReadFile(filepath.Join(folder, checksums))
			assert.NoError(t, err)
			for _, want := range tt.want {
				assert.Contains(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  "+want)
			}
		})
	}

}

func TestPipeSkipTrue(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				Disable: true,
			},
		},
	)
	err = Pipe{}.Run(ctx)
	testlib.AssertSkipped(t, err)
	assert.EqualError(t, err, `checksum.disable is set`)
}

func TestPipeFileNotExist(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = context.New(
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
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/nope: no such file or directory")
}

func TestPipeInvalidNameTemplate(t *testing.T) {
	binFile, err := ioutil.TempFile("", "goreleasertest-bin")
	assert.NoError(t, err)
	_, err = binFile.WriteString("fake artifact")
	assert.NoError(t, err)

	for template, eerr := range map[string]string{
		"{{ .Pro }_checksums.txt": `template: tmpl:1: unexpected "}" in operand`,
		"{{.Env.NOPE}}":           `template: tmpl:1:6: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`,
	} {
		t.Run(template, func(tt *testing.T) {
			folder, err := ioutil.TempDir("", "goreleasertest")
			assert.NoError(tt, err)
			var ctx = context.New(
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
			assert.Error(tt, err)
			assert.Equal(tt, eerr, err.Error())
		})
	}
}

func TestPipeCouldNotOpenChecksumsTxt(t *testing.T) {
	binFile, err := ioutil.TempFile("", "goreleasertest-bin")
	assert.NoError(t, err)
	_, err = binFile.WriteString("fake artifact")
	assert.NoError(t, err)

	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var file = filepath.Join(folder, "checksums.txt")
	assert.NoError(t, ioutil.WriteFile(file, []byte("some string"), 0000))
	var ctx = context.New(
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/checksums.txt: permission denied")
}

func TestPipeWhenNoArtifacts(t *testing.T) {
	var ctx = &context.Context{}
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Len(t, ctx.Artifacts.List(), 0)
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Checksum: config.Checksum{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(
		t,
		"{{ .ProjectName }}_{{ .Version }}_checksums.txt",
		ctx.Config.Checksum.NameTemplate,
	)
	assert.Equal(t, "sha256", ctx.Config.Checksum.Algorithm)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "checksums.txt", ctx.Config.Checksum.NameTemplate)
}

// TODO: add tests for LinuxPackage and UploadableSourceArchive
