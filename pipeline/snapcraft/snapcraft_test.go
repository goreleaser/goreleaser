package snapcraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.Description())
}

func TestRunPipeMissingInfo(t *testing.T) {
	for eerr, snap := range map[error]config.Snapcraft{
		ErrNoSummary: {
			Description: "dummy desc",
		},
		ErrNoDescription: {
			Summary: "dummy summary",
		},
		nil: {}, // should skip instead of error
	} {
		t.Run(fmt.Sprintf("testing if %v happens", eerr), func(t *testing.T) {
			var assert = assert.New(t)
			var ctx = &context.Context{
				Config: config.Project{
					Snapcraft: snap,
				},
			}
			assert.Equal(eerr, Pipe{}.Run(ctx))
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
				Name:             "foo",
				FilenameTemplate: "{{.ProjectName}}_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
			},
		},
	}
	for _, plat := range []string{"linuxamd64", "linux386", "darwinamd64", "linuxarm64", "linuxarmhf"} {
		var folder = "mybin_" + plat
		assert.NoError(os.Mkdir(filepath.Join(dist, folder), 0755))
		var binPath = filepath.Join(dist, folder, "mybin")
		_, err = os.Create(binPath)
		assert.NoError(err)
		ctx.AddBinary(plat, folder, "mybin", binPath)
	}
	assert.NoError(Pipe{}.Run(ctx))
	// TODO: assert file exist with the correct name
}

func TestRunPipeWithPlugsAndDaemon(t *testing.T) {
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
				Apps: map[string]config.SnapcraftAppMetadata{
					"mybin": {
						Plugs:  []string{"home", "network"},
						Daemon: "simple",
					},
				},
			},
		},
	}
	for _, plat := range []string{"linuxamd64", "linux386", "darwinamd64", "linuxarm64", "linuxarmhf"} {
		var folder = "mybin_" + plat
		assert.NoError(os.Mkdir(filepath.Join(dist, folder), 0755))
		var binPath = filepath.Join(dist, folder, "mybin")
		_, err = os.Create(binPath)
		ctx.AddBinary(plat, folder, "mybin", binPath)
	}
	assert.NoError(Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "mybin_linuxamd64", "prime", "meta", "snap.yaml"))
	assert.NoError(err)
	var snapcraftMetadata SnapcraftMetadata
	err = yaml.Unmarshal(yamlFile, &snapcraftMetadata)
	assert.NoError(err)
	assert.Equal(snapcraftMetadata.Apps["mybin"].Plugs, []string{"home", "network"})
	assert.Equal(snapcraftMetadata.Apps["mybin"].Daemon, "simple")
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
