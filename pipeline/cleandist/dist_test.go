package cleandist

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
)

func TestDistDoesNotExist(t *testing.T) {
	var assert = assert.New(t)
	assert.NoError(
		Pipe{}.Run(
			&context.Context{
				Config: config.Project{
					Dist: "/wtf-this-shouldnt-exist",
				},
			},
		),
	)
}

func TestPopulatedDistExists(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "disttest")
	assert.NoError(err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	_, err = os.Create(filepath.Join(dist, "mybin"))
	assert.NoError(err)
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
		},
	}
	assert.Error(Pipe{}.Run(ctx))
	ctx.RmDist = true
	assert.NoError(Pipe{}.Run(ctx))
	_, err = os.Stat(dist)
	assert.False(os.IsExist(err))
}

func TestEmptyDistExists(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "disttest")
	assert.NoError(err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	var ctx = &context.Context{
		Config: config.Project{
			Dist: dist,
		},
	}
	assert.NoError(Pipe{}.Run(ctx))
	_, err = os.Stat(dist)
	assert.False(os.IsExist(err))
}

func TestDescription(t *testing.T) {
	var assert = assert.New(t)
	assert.NotEmpty(Pipe{}.Description())
}
