package snapcraft

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/internal/yaml"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
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
			ctx := &context.Context{
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
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate:     "foo_{{.Arch}}",
				Summary:          "test summary {{.ProjectName}}",
				Description:      "test description {{.ProjectName}}",
				Publish:          true,
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
			{
				NameTemplate:     "foo_and_bar_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
				Publish:          true,
				Builds:           []string{"foo", "bar"},
				ChannelTemplates: []string{"stable"},
			},
			{
				NameTemplate:     "bar_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
				Publish:          true,
				Builds:           []string{"bar"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	addBinaries(t, ctx, "bar", filepath.Join(dist, "bar"))
	require.NoError(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	require.Len(t, list, 9)
}

func TestBadTemolate(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate:     "foo_{{.Arch}}",
				Publish:          true,
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))

	t.Run("description", func(t *testing.T) {
		ctx.Config.Snapcrafts[0].Description = "{{.Bad}}"
		ctx.Config.Snapcrafts[0].Summary = "summary"
		require.Error(t, Pipe{}.Run(ctx))
	})

	t.Run("summary", func(t *testing.T) {
		ctx.Config.Snapcrafts[0].Description = "description"
		ctx.Config.Snapcrafts[0].Summary = "{{.Bad}}"
		require.Error(t, Pipe{}.Run(ctx))
	})
}

func TestRunPipeInvalidNameTemplate(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "foo",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate:     "foo_{{.Arch}",
				Summary:          "test summary",
				Description:      "test description",
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.EqualError(t, Pipe{}.Run(ctx), `template: tmpl:1: unexpected "}" in operand`)
}

func TestRunPipeWithName(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate:     "foo_{{.Arch}}",
				Name:             "testsnapname",
				Base:             "core18",
				License:          "MIT",
				Summary:          "test summary",
				Description:      "test description",
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "testsnapname", metadata.Name)
	require.Equal(t, "core18", metadata.Base)
	require.Equal(t, "MIT", metadata.License)
	require.Equal(t, "foo", metadata.Apps["testsnapname"].Command)
}

func TestRunPipeMetadata(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				Name:         "testprojectname",
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Layout: map[string]config.SnapcraftLayoutMetadata{
					"/etc/testprojectname": {Bind: "$SNAP_DATA/etc"},
				},
				Apps: map[string]config.SnapcraftAppMetadata{
					"before-foo": {
						Before:  []string{"foo"},
						Command: "foo",
						Daemon:  "notify",
					},
					"after-foo": {
						After:   []string{"foo"},
						Command: "foo",
						Daemon:  "notify",
					},
					"foo": {
						Args:         "--foo --bar",
						Adapter:      "foo_adapter",
						Aliases:      []string{"dummy_alias"},
						Autostart:    "foobar.desktop",
						BusName:      "foo_busname",
						CommandChain: []string{"foo_cmd_chain"},
						CommonID:     "foo_common_id",
						Completer:    "", // Separately tested in TestCompleter
						Daemon:       "simple",
						Desktop:      "foo_desktop",
						Environment: map[string]interface{}{
							"foo": "bar",
						},
						Extensions:  []string{"foo_extension"},
						InstallMode: "disable",
						Passthrough: map[string]interface{}{
							"planet": "saturn",
						},
						Plugs:            []string{"home", "network", "network-bind", "personal-files"},
						PostStopCommand:  "foo",
						RefreshMode:      "endure",
						ReloadCommand:    "foo",
						RestartCondition: "always",
						RestartDelay:     "42ms",
						Slots:            []string{"foo_slot"},
						Sockets: map[string]interface{}{
							"sock": map[string]interface{}{
								"listen-stream": "$SNAP_COMMON/socket",
								"socket-group":  "socket-group",
								"socket-mode":   0o640,
							},
						},
						StartTimeout:    "43ms",
						StopCommand:     "foo",
						StopMode:        "sigterm",
						StopTimeout:     "44ms",
						Timer:           "00:00-24:00/24",
						WatchdogTimeout: "45ms",
					},
				},
				Plugs: map[string]interface{}{
					"personal-files": map[string]interface{}{
						"read": []string{"$HOME/test"},
					},
				},
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, map[string]AppMetadata{
		"before-foo": {
			Before:  []string{"foo"},
			Command: "foo",
			Daemon:  "notify",
		},
		"after-foo": {
			After:   []string{"foo"},
			Command: "foo",
			Daemon:  "notify",
		},
		"foo": {
			Adapter:      "foo_adapter",
			Aliases:      []string{"dummy_alias"},
			Autostart:    "foobar.desktop",
			BusName:      "foo_busname",
			Command:      "foo --foo --bar",
			CommandChain: []string{"foo_cmd_chain"},
			CommonID:     "foo_common_id",
			Completer:    "",
			Daemon:       "simple",
			Desktop:      "foo_desktop",
			Environment: map[string]interface{}{
				"foo": "bar",
			},
			Extensions:  []string{"foo_extension"},
			InstallMode: "disable",
			Passthrough: map[string]interface{}{
				"planet": "saturn",
			},
			Plugs:            []string{"home", "network", "network-bind", "personal-files"},
			PostStopCommand:  "foo",
			RefreshMode:      "endure",
			ReloadCommand:    "foo",
			RestartCondition: "always",
			RestartDelay:     "42ms",
			Slots:            []string{"foo_slot"},
			Sockets: map[string]interface{}{
				"sock": map[string]interface{}{
					"listen-stream": "$SNAP_COMMON/socket",
					"socket-group":  "socket-group",
					"socket-mode":   0o640,
				},
			},
			StartTimeout:    "43ms",
			StopCommand:     "foo",
			StopMode:        "sigterm",
			StopTimeout:     "44ms",
			Timer:           "00:00-24:00/24",
			WatchdogTimeout: "45ms",
		},
	}, metadata.Apps)
	require.Equal(t, map[string]interface{}{"read": []interface{}{"$HOME/test"}}, metadata.Plugs["personal-files"])
	require.Equal(t, "$SNAP_DATA/etc", metadata.Layout["/etc/testprojectname"].Bind)
}

func TestNoSnapcraftInPath(t *testing.T) {
	path := os.Getenv("PATH")
	defer func() {
		require.NoError(t, os.Setenv("PATH", path))
	}()
	require.NoError(t, os.Setenv("PATH", ""))
	ctx := context.New(config.Project{
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
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"foo": {
						Daemon: "simple",
						Args:   "",
					},
				},
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "foo", metadata.Apps["foo"].Command)
}

func TestCompleter(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"foo": {
						Daemon:    "simple",
						Args:      "",
						Completer: "testdata/foo-completer.bash",
					},
				},
				Builds:           []string{"foo", "bar"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	addBinaries(t, ctx, "bar", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "foo", metadata.Apps["foo"].Command)
	require.Equal(t, "testdata/foo-completer.bash", metadata.Apps["foo"].Completer)
}

func TestCommand(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Apps: map[string]config.SnapcraftAppMetadata{
					"foo": {
						Daemon:  "simple",
						Args:    "--bar custom command",
						Command: "foo",
					},
				},
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "foo --bar custom command", metadata.Apps["foo"].Command)
}

func TestExtraFile(t *testing.T) {
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := context.New(config.Project{
		ProjectName: "testprojectname",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate: "foo_{{.Arch}}",
				Summary:      "test summary",
				Description:  "test description",
				Files: []config.SnapcraftExtraFiles{
					{
						Source:      "testdata/extra-file.txt",
						Destination: "a/b/c/extra-file.txt",
						Mode:        0o755,
					},
					{
						Source: "testdata/extra-file-2.txt",
					},
				},
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	})
	ctx.Git.CurrentTag = "v1.2.3"
	ctx.Version = "1.2.3"
	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))

	apath := filepath.Join(dist, "foo_amd64", "prime", "a", "b", "c", "extra-file.txt")
	bpath := filepath.Join(dist, "foo_amd64", "prime", "testdata", "extra-file-2.txt")
	requireEqualFileConents(t, "testdata/extra-file.txt", apath)
	requireEqualFileConents(t, "testdata/extra-file-2.txt", bpath)
}

func TestDefault(t *testing.T) {
	ctx := context.New(config.Project{
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
	ctx := context.New(config.Project{})
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
		Extra: map[string]interface{}{
			releasesExtra: []string{"stable", "candidate"},
		},
	})
	err := Pipe{}.Publish(ctx)
	require.Contains(t, err.Error(), "failed to push nope.snap package")
}

func TestPublishSkip(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.SkipPublish = true
	ctx.Artifacts.Add(&artifact.Artifact{
		Name:   "mybin",
		Path:   "nope.snap",
		Goarch: "amd64",
		Goos:   "linux",
		Type:   artifact.PublishableSnapcraft,
		Extra: map[string]interface{}{
			releasesExtra: []string{"stable"},
		},
	})
	testlib.AssertSkipped(t, Pipe{}.Publish(ctx))
}

func TestDefaultSet(t *testing.T) {
	ctx := context.New(config.Project{
		Snapcrafts: []config.Snapcraft{
			{
				ID:           "devel",
				NameTemplate: "foo",
				Grade:        "devel",
			},
			{
				ID:           "stable",
				NameTemplate: "bar",
				Grade:        "stable",
			},
		},
	})
	require.NoError(t, Pipe{}.Default(ctx))
	require.Equal(t, "foo", ctx.Config.Snapcrafts[0].NameTemplate)
	require.Equal(t, []string{"edge", "beta"}, ctx.Config.Snapcrafts[0].ChannelTemplates)
	require.Equal(t, []string{"edge", "beta", "candidate", "stable"}, ctx.Config.Snapcrafts[1].ChannelTemplates)
}

func Test_processChannelsTemplates(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					ID: "default",
				},
			},
			Snapcrafts: []config.Snapcraft{
				{
					Name: "mybin",
					ChannelTemplates: []string{
						"{{.Major}}.{{.Minor}}/stable",
						"stable",
					},
				},
			},
		},
	}

	ctx.SkipPublish = true
	ctx.Env = map[string]string{
		"FOO": "123",
	}
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{
		CurrentTag: "v1.0.0",
		Commit:     "a1b2c3d4",
	}
	ctx.Semver = context.Semver{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

	require.NoError(t, Pipe{}.Default(ctx))

	snap := ctx.Config.Snapcrafts[0]
	require.Equal(t, "mybin", snap.Name)

	channels, err := processChannelsTemplates(ctx, snap)
	require.NoError(t, err)
	require.Equal(t, []string{
		"1.0/stable",
		"stable",
	}, channels)
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	t.Helper()
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm6"} {
			folder := goos + goarch
			require.NoError(t, os.MkdirAll(filepath.Join(dist, folder), 0o755))
			binPath := filepath.Join(dist, folder, name)
			f, err := os.Create(binPath)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:   "subdir/" + name,
				Path:   binPath,
				Goarch: goarch,
				Goos:   goos,
				Type:   artifact.Binary,
				Extra: map[string]interface{}{
					artifact.ExtraID: name,
				},
			})
		}
	}
}

func TestSeveralSnapssWithTheSameID(t *testing.T) {
	ctx := &context.Context{
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

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			Snapcrafts: []config.Snapcraft{
				{},
			},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func requireEqualFileConents(tb testing.TB, a, b string) {
	tb.Helper()
	eq, err := gio.EqualFileContents(a, b)
	require.NoError(tb, err)
	require.True(tb, eq, "%s != %s", a, b)
}
