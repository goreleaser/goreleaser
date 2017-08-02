package snapcraft

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

func TestRunPipeMissingInfo(t *testing.T) {
	for name, snap := range map[string]config.Snapcraft{
		"missing summary": {
			Description: "dummy desc",
		},
		"missing description": {
			Summary: "dummy summary",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var assert = assert.New(t)
			var ctx = &context.Context{
				Config: config.Project{
					Snapcraft: snap,
				},
			}
			assert.NoError(Pipe{}.Run(ctx))
		})
	}
}

func TestRunPipe(t *testing.T) {
	var assert = assert.New(t)
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(os.Mkdir(dist, 0755))
	assert.NoError(err)
	var ctx = &context.Context{
		Version: "testversion",
		Config: config.Project{
			ProjectName: "mybin",
			Dist:        dist,
			Snapcraft: config.Snapcraft{
				Summary:     "test summary",
				Description: "test description",
			},
		},
	}
	for _, plat := range []string{"linuxamd64", "linux386", "darwinamd64"} {
		var folder = "mybin_" + plat
		assert.NoError(os.Mkdir(filepath.Join(dist, folder), 0755))
		var binPath = filepath.Join(dist, folder, "mybin")
		_, err = os.Create(binPath)
		ctx.AddBinary(plat, folder, "mybin", binPath)
	}
	assert.NoError(Pipe{}.Run(ctx))
}

func TestNoSnapcraftInPath(t *testing.T) {
	var assert = assert.New(t)
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(os.Setenv("PATH", path))
	}()
	assert.NoError(os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Config: config.Project{
			Snapcraft: config.Snapcraft{
				Summary:     "dummy",
				Description: "dummy",
			},
		},
	}
	assert.EqualError(Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}
