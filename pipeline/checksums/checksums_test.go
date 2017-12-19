package checksums

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestPipe(t *testing.T) {
	var binary = "binary"
	var checksums = binary + "_bar_checksums.txt"
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
			},
		},
	)
	ctx.Env = map[string]string{"FOO": "bar"}
	ctx.Artifacts.Add(artifact.Artifact{
		Name: binary,
		Path: file,
		Type: artifact.UploadableBinary,
	})
	ctx.Artifacts.Add(artifact.Artifact{
		Name: binary + ".tar.gz",
		Path: file,
		Type: artifact.UploadableArchive,
	})
	assert.NoError(t, Pipe{}.Run(ctx))
	var artifacts []string
	for _, a := range ctx.Artifacts.List() {
		artifacts = append(artifacts, a.Name)
	}
	assert.Contains(t, artifacts, checksums, binary)
	bts, err := ioutil.ReadFile(filepath.Join(folder, checksums))
	assert.NoError(t, err)
	assert.Contains(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  binary")
	assert.Contains(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  binary.tar.gz")
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
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "nope",
		Path: "/nope",
		Type: artifact.UploadableBinary,
	})
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/nope: no such file or directory")
}

func TestPipeInvalidNameTemplate(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = context.New(
		config.Project{
			Dist:        folder,
			ProjectName: "name",
			Checksum: config.Checksum{
				NameTemplate: "{{ .Pro }_checksums.txt",
			},
		},
	)
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "whatever",
		Type: artifact.UploadableBinary,
	})
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Equal(t, `template: checksums:1: unexpected "}" in operand`, err.Error())
}

func TestPipeCouldNotOpenChecksumsTxt(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var file = filepath.Join(folder, "checksums.txt")
	assert.NoError(t, ioutil.WriteFile(file, []byte("some string"), 0000))
	var ctx = context.New(
		config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	)
	ctx.Artifacts.Add(artifact.Artifact{
		Name: "whatever",
		Type: artifact.UploadableBinary,
	})
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/checksums.txt: permission denied")
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
