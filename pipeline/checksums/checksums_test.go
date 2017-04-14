package checksums

import (
	"io/ioutil"
	"os"
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
	folder, err := ioutil.TempDir("", "gorelasertest")
	assert.NoError(err)
	file, err := os.OpenFile(
		filepath.Join(folder, "binary"),
		os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	assert.NoError(err)
	_, err = file.WriteString("some string")
	assert.NoError(err)
	assert.NoError(file.Close())
	var ctx = &context.Context{
		Config: config.Project{
			Dist: folder,
		},
	}
	ctx.AddArtifact(file.Name())
	assert.NoError(Pipe{}.Run(ctx))
	assert.Contains(ctx.Artifacts, "binary.checksums", "binary")
	bts, err := ioutil.ReadFile(filepath.Join(folder, "binary.checksums"))
	assert.NoError(err)
	assert.Contains(string(bts), "61d034473102d7dac305902770471fd50f4c5b26f6831a56dd90b5184b3c30fc	binary")
}
