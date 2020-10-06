package snapcraft

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
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
			require.Equal(t, eerr, Pipe{}.Run(ctx))
		})
	}
}

func TestRunPipe(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
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
	require.NoError(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	require.Len(t, list, 9)
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
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
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithName(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
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
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "testsnapname", metadata.Name)
	require.Equal(t, "core18", metadata.Base)
	require.Equal(t, "MIT", metadata.License)
	require.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeWithBinaryInDir(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
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
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "testsnapname", metadata.Name)
	require.Equal(t, "", metadata.Base)
	require.Equal(t, "", metadata.License)
	require.Equal(t, "mybin", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeMetadata(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
	var dist = filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0755))
	require.NoError(t, err)
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
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["mybin"].Plugs)
	require.Equal(t, "simple", metadata.Apps["mybin"].Daemon)
	require.Equal(t, "mybin --foo --bar", metadata.Apps["mybin"].Command)
	require.Equal(t, []string{"home", "network", "personal-files"}, metadata.Apps["mybin"].Plugs)
	require.Equal(t, "simple", metadata.Apps["mybin"].Daemon)
	require.Equal(t, "mybin --foo --bar", metadata.Apps["mybin"].Command)
	require.Equal(t, map[interface{}]interface{}(map[interface{}]interface{}{"read": []interface{}{"$HOME/test"}}), metadata.Plugs["personal-files"])
}

func TestNoSnapcraftInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		require.NoError(t, os.Setenv("PATH", path))
	}()
	require.NoError(t, os.Setenv("PATH", ""))
	var ctx = context.New(config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				Summary:     "dummy",
				Description: "dummy",
			},
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), ErrNoSnapcraft.Error())
}

func TestRunNoArguments(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
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
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "mybin", metadata.Apps["mybin"].Command)
}

func TestCompleter(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
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
						Completer: "testdata/mybin-completer.bash",
					},
				},
				Builds: []string{"foo", "bar"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "v1.2.3"
	addBinaries(t, ctx, "foo", dist, "mybin")
	addBinaries(t, ctx, "bar", dist, "mybin")
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := ioutil.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "mybin", metadata.Apps["mybin"].Command)
	require.Equal(t, "testdata/mybin-completer.bash", metadata.Apps["mybin"].Completer)
}

func TestCommand(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
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
						Daemon:  "simple",
						Args:    "",
						Command: "custom command",
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
	require.Equal(t, "custom command", metadata.Apps["mybin"].Command)
}

func TestExtraFile(t *testing.T) {
	folder, err := ioutil.TempDir("", "archivetest")
	require.NoError(t, err)
	defer os.RemoveAll(folder)
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
						Daemon:  "simple",
						Args:    "",
						Command: "custom command",
					},
				},
				Files: []config.SnapcraftExtraFiles{
					{
						Source:      "testdata/extra-file.txt",
						Destination: "a/b/c/extra-file.txt",
						Mode:        0755,
					},
					{
						Source: "testdata/extra-file-2.txt",
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

	srcFile, err := os.Stat("testdata/extra-file.txt")
	require.NoError(t, err)
	destFile, err := os.Stat(filepath.Join(dist, "foo_amd64", "prime", "a", "b", "c", "extra-file.txt"))
	require.NoError(t, err)
	require.Equal(t, inode(srcFile), inode(destFile))
	require.Equal(t, destFile.Mode(), os.FileMode(0755))

	srcFile, err = os.Stat("testdata/extra-file-2.txt")
	require.NoError(t, err)
	destFileWithDefaults, err := os.Stat(filepath.Join(dist, "foo_amd64", "prime", "testdata", "extra-file-2.txt"))
	require.NoError(t, err)
	require.Equal(t, destFileWithDefaults.Mode(), os.FileMode(0644))
	require.Equal(t, inode(srcFile), inode(destFileWithDefaults))
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
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, defaultNameTemplate, ctx.Config.Snapcrafts[0].NameTemplate)
	require.Equal(t, []string{"foo"}, ctx.Config.Snapcrafts[0].Builds)
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
	require.Contains(t, err.Error(), "failed to push nope.snap package")
}

func TestPublishSkip(t *testing.T) {
	var ctx = context.New(config.Project{})
	ctx.SkipPublish = true
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestDefaultSet(t *testing.T) {
	var ctx = context.New(config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Snapcrafts[0].NameTemplate)
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist, dest string) {
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm6"} {
			var folder = goos + goarch
			require.NoError(t, os.MkdirAll(filepath.Join(dist, folder), 0755))
			var binPath = filepath.Join(dist, folder, name)
			_, err := os.Create(binPath)
			require.NoError(t, err)
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

func Test_isValidArch(t *testing.T) {
	tests := []struct {
		arch string
		want bool
	}{
		{"s390x", true},
		{"ppc64el", true},
		{"arm64", true},
		{"armhf", true},
		{"amd64", true},
		{"i386", true},
		{"mips", false},
		{"armel", false},
	}
	for _, tt := range tests {
		t.Run(tt.arch, func(t *testing.T) {
			require.Equal(t, tt.want, isValidArch(tt.arch))
		})
	}
}

func inode(info os.FileInfo) uint64 {
	stat := info.Sys().(*syscall.Stat_t)
	return stat.Ino
}
