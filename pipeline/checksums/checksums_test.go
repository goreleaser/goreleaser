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
	var assert = assert.New(t)
	var checksums = "binary_checksums.txt"
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var file = filepath.Join(folder, "binary")
	assert.NoError(ioutil.WriteFile(file, []byte("some string"), 0644))
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
			Build: config.Build{
				Binary: "binary",
			},
		},
	}
	ctx.AddArtifact(file)
	assert.NoError(Pipe{}.Run(ctx))
	assert.Contains(ctx.Artifacts, checksums, "binary")
	bts, err := ioutil.ReadFile(filepath.Join(folder, checksums))
	assert.NoError(err)
	assert.Contains(string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc	binary")
}

func TestPipeFileNotExist(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
		},
	}
	ctx.AddArtifact("nope")
	assert.Error(Pipe{}.Run(ctx))
}

func TestPipeFileCantBeWritten(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "goreleasertest")
	assert.NoError(err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
		},
	}
	ctx.AddArtifact("nope")
	assert.Error(Pipe{}.Run(ctx))
}
