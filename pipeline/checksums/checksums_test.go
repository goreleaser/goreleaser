package checksums

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestPipe(t *testing.T) {
	var binary = "binary"
	var checksums = binary + "_checksums.txt"
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var file = filepath.Join(folder, binary)
	assert.NoError(t, ioutil.WriteFile(file, []byte("some string"), 0644))
	var ctx = &context.Context{
		Config: config.Project{
			Dist:        folder,
			ProjectName: binary,
			Checksum: config.Checksum{
				NameTemplate: "{{ .ProjectName }}_checksums.txt",
			},
		},
	}
	ctx.AddArtifact(file)
	assert.NoError(t, Pipe{}.Run(ctx))
	assert.Contains(t, ctx.Artifacts, checksums, binary)
	bts, err := ioutil.ReadFile(filepath.Join(folder, checksums))
	assert.NoError(t, err)
	assert.Equal(t, string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc  binary\n")
}

func TestPipeFileNotExist(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	}
	ctx.AddArtifact("nope")
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/nope: no such file or directory")
}

func TestPipeInvalidNameTemplate(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist:        folder,
			ProjectName: "name",
			Checksum: config.Checksum{
				NameTemplate: "{{ .Pro }_checksums.txt",
			},
		},
	}
	ctx.AddArtifact("whatever")
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Equal(t, `template: name:1: unexpected "}" in operand`, err.Error())
}

func TestPipeCouldNotOpenChecksumsTxt(t *testing.T) {
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(t, err)
	var file = filepath.Join(folder, "checksums.txt")
	assert.NoError(t, ioutil.WriteFile(file, []byte("some string"), 0000))
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
			Checksum: config.Checksum{
				NameTemplate: "checksums.txt",
			},
		},
	}
	ctx.AddArtifact("nope")
	err = Pipe{}.Run(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/checksums.txt: permission denied")
}
