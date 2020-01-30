package snapcraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestRunPipeMissingInfo(t *testing.T) {
	for eerr, snap := range map[error]config.Snapcraft{
		ErrNoSummary: {
			Description: "dummy desc",
		},
		ErrNoDescription: {
			Summary: "dummy summary",
		},
		pipe.Skip("no summary nor description were provided"): {},
	} {
		t.Run(fmt.Sprintf("testing if %v happens", eerr), func(t *testing.T) {
			var ctx = &context.Context{
				Config: config.Project{
					Snapcrafts: []config.Snapcraft{
						snap,
					},
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
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Publish:      true,
				Builds:       []string{"foo"},
			},
			{
				NameTemplate: "foo_and_bar_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Publish:      true,
				Builds:       []string{"foo", "bar"},
			},
			{
				NameTemplate: "bar_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Publish:      true,
				Builds:       []string{"bar"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"), "foo")
	addBinaries(t, ctx, "bar", filepath.Join(dist, "bar"), "bar")
	assert.NoError(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	assert.Len(t, list, 9)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}",
				Summary:      "test summary",
				Description:  "test description",
				Builds:       []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	assert.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithName(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Name:         "testsnapname",
				Base:         "core18",
				License:      "MIT",
				Summary:      "test summary",
				Description:  "test description",
				Builds:       []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, "testsnapname", metadata.Name)
	assert.Equal(t, "core18", metadata.Base)
	assert.Equal(t, "MIT", metadata.License)
	assert.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeWithBinaryInDir(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Name:         "testsnapname",
				Summary:      "test summary",
				Description:  "test description",
				Builds:       []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "bin/mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, "testsnapname", metadata.Name)
	assert.Equal(t, "", metadata.Base)
	assert.Equal(t, "", metadata.License)
	assert.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeMetadata(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				Name:         "testprojectname",
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"mybin": {
						Plugs:  []string{"home", "network", "personal-files"},
						Daemon: "simple",
						Args:   "--foo --bar",
					},
				},
				Plugs: map[string]interface{}{
					"personal-files": map[string]interface{}{
						"read": []string{"$HOME/test"},
					},
				},
				Builds: []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["mybin"].Plugs)
	assert.Equal(t, "simple", metadata.Apps["mybin"].Daemon)
	assert.Equal(t, "mybin --foo --bar", metadata.Apps["mybin"].Command)
	assert.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["mybin"].Plugs)
	assert.Equal(t, "simple", metadata.Apps["mybin"].Daemon)
	assert.Equal(t, "mybin --foo --bar", metadata.Apps["mybin"].Command)
	assert.Equal(t, map[interface{}]interface{}(map[interface{}]interface{}{"read": []interface{}{"$HOME/test"}}), metadata.Plugs["personal-files"])
}

func TestNoSnapcraftInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = context.New(config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				Summary:     "dummy",
				Description: "dummy",
			},
		},
	})
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}

func TestRunNoArguments(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	assert.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	assert.NoError(t, os.Mkdir(dist, 0755))
	assert.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"mybin": {
						Daemon: "simple",
						Args:   "",
					},
				},
				Builds: []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	assert.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	assert.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	assert.NoError(t, err)
	assert.Equal(t, "mybin", metadata.Apps["mybin"].Command)
}

func TestCompleter(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
	var ctx = context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"mybin": {
						Daemon:    "simple",
						Args:      "",
						Completer: "mybin-completer.bash",
					},
				},
				Builds: []string{"foo"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	assert.Equal(t, "mybin", metadata.Apps["mybin"].Command)
	assert.Equal(t, "mybin-completer.bash", metadata.Apps["mybin"].Completer)
}

func TestDefault(t *testing.T) {
	var ctx = context.New(config.Project{
		Builds: []config.Build{
			{
				ID: "foo",
			},
		},
		Snapcrafts: []config.Snapcraft{
			{},
		},
	})
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, defaultNameTemplate, ctx.Config.Snapcrafts[0].NameTemplate)
	assert.Equal(t, []string{"foo"}, ctx.Config.Snapcrafts[0].Builds)
}

func TestPublish(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
	})
	err := Pipe{}.Publish(ctx)
	assert.Contains(t, err.Error(), "failed to push nope.snap package")
}

func TestDefaultSet(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo",
			},
		},
	})
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Equal(t, "foo", ctx.Config.Snapcrafts[0].NameTemplate)
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist, dest string) {
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm6"} {
			var folder = goos + goarch
			assert.NoError(t, os.MkdirAll(filepath.Join(dist, folder), 0755))
			var binPath = filepath.Join(dist, folder, name)
			_, err := os.Create(binPath)
			assert.NoError(t, err)
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   dest,
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					"ID": name,
				},
			})
		}
	}
}

func TestSeveralSnapssWithTheSameID(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Snapcrafts: []config.Snapcraft{
				{
					ID: "a",
				},
				{
					ID: "a",
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 snapcrafts with the ID 'a', please fix your config")
}
