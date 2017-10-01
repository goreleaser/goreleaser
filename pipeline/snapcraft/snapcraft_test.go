package snapcraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
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
		pipeline.Skip("no summary nor description were provided"): {},
	} {
		t.Run(fmt.Sprintf("testing if %v happens", eerr), func(t *testing.T) {
			var ctx = &context.Context{
				Config: config.Project{
					Snapcraft: snap,
				},
			}
			assert.Equal(t, eerr, Pipe{}.Run(ctx))
		})
	}
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
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
	addBinaries(t, ctx, "mybin", dist)
	assert.NoError(t, Pipe{}.Run(ctx))
}

func TestRunPipeWithName(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = &context.Context{
		Version: "testversion",
		Config: config.Project{
			ProjectName: "testprojectname",
			Dist:        dist,
			Snapcraft: config.Snapcraft{
				Name:        "testsnapname",
				Summary:     "test summary",
				Description: "test description",
			},
		},
	}
	addBinaries(t, ctx, "testprojectname", dist)
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "testprojectname_linuxamd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, metadata.Name, "testsnapname")
}

func TestRunPipeWithPlugsAndDaemon(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
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
	addBinaries(t, ctx, "mybin", dist)
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "mybin_linuxamd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, metadata.Apps["mybin"].Plugs, []string{"home", "network"})
	assert.Equal(t, metadata.Apps["mybin"].Daemon, "simple")
}

func TestNoSnapcraftInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Config: config.Project{
			Snapcraft: config.Snapcraft{
				Summary:     "dummy",
				Description: "dummy",
			},
		},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	for _, plat := range []string{
		"linuxamd64",
		"linux386",
		"darwinamd64",
		"linuxarm64",
		"linuxarm6",
		"linuxwtf",
	} {
		var folder = name + "_" + plat
		assert.NoError(t, os.Mkdir(filepath.Join(dist, folder), 0755))
		var binPath = filepath.Join(dist, folder, name)
		_, err := os.Create(binPath)
		assert.NoError(t, err)
		ctx.AddBinary(plat, folder, name, binPath)
	}
}
