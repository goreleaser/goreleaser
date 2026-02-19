//go:build integration

package snapcraft

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gio"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/internal/yaml"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRunPipe(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		ProjectName: "mybin",
		Dist:        dist,
		Snapcrafts: []config.Snapcraft{
			{
				NameTemplate:     "foo_{{.Arch}}",
				Summary:          "test summary {{.ProjectName}}",
				Description:      "test description {{.ProjectName}}",
				Publish:          true,
				IDs:              []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
			{
				NameTemplate:     "foo_and_bar_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
				Publish:          true,
				IDs:              []string{"foo", "bar"},
				ChannelTemplates: []string{"stable"},
			},
			{
				NameTemplate:     "bar_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
				Publish:          true,
				IDs:              []string{"bar"},
				ChannelTemplates: []string{"stable"},
			},
			{
				NameTemplate:     "bar_{{.Arch}}",
				Summary:          "test summary",
				Description:      "test description",
				Publish:          true,
				IDs:              []string{"bar"},
				ChannelTemplates: []string{"stable"},
				Disable:          "{{.Env.SKIP}}",
			},
		},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"), testctx.WithEnv(map[string]string{"SKIP": "true"}))

	addBinaries(t, ctx, "foo", filepath.Join(dist, "foo"))
	addBinaries(t, ctx, "bar", filepath.Join(dist, "bar"))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	list := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	require.Len(t, list, 9)
}

func TestIntegrationBadTemplate(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

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

func TestIntegrationRunPipeInvalidNameTemplate(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
}

func TestIntegrationRunPipeWithName(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

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

func TestIntegrationRunPipeMetadata(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
						Completer:    "",
						Daemon:       "simple",
						Desktop:      "foo_desktop",
						Environment: map[string]any{
							"foo": "bar",
						},
						Extensions:  []string{"foo_extension"},
						InstallMode: "disable",
						Passthrough: map[string]any{
							"planet": "saturn",
						},
						Plugs:            []string{"home", "network", "network-bind", "personal-files"},
						PostStopCommand:  "foo",
						RefreshMode:      "endure",
						ReloadCommand:    "foo",
						RestartCondition: "always",
						RestartDelay:     "42ms",
						Slots:            []string{"foo_slot"},
						Sockets: map[string]any{
							"sock": map[string]any{
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
				Plugs: map[string]any{
					"personal-files": map[string]any{
						"read": []string{"$HOME/test"},
					},
				},
				Builds:           []string{"foo"},
				ChannelTemplates: []string{"stable"},
			},
		},
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

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
			Environment: map[string]any{
				"foo": "bar",
			},
			Extensions:  []string{"foo_extension"},
			InstallMode: "disable",
			Passthrough: map[string]any{
				"planet": "saturn",
			},
			Plugs:            []string{"home", "network", "network-bind", "personal-files"},
			PostStopCommand:  "foo",
			RefreshMode:      "endure",
			ReloadCommand:    "foo",
			RestartCondition: "always",
			RestartDelay:     "42ms",
			Slots:            []string{"foo_slot"},
			Sockets: map[string]any{
				"sock": map[string]any{
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
	require.Equal(t, map[string]any{"read": []any{"$HOME/test"}}, metadata.Plugs["personal-files"])
	require.Equal(t, "$SNAP_DATA/etc", metadata.Layout["/etc/testprojectname"].Bind)
}

func TestIntegrationRunNoArguments(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "foo", metadata.Apps["foo"].Command)
}

func TestIntegrationCompleter(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

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

func TestIntegrationCommand(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))
	yamlFile, err := os.ReadFile(filepath.Join(dist, "foo_amd64", "prime", "meta", "snap.yaml"))
	require.NoError(t, err)
	var metadata Metadata
	err = yaml.Unmarshal(yamlFile, &metadata)
	require.NoError(t, err)
	require.Equal(t, "foo --bar custom command", metadata.Apps["foo"].Command)
}

func TestIntegrationExtraFile(t *testing.T) {
	testlib.SkipIfWindows(t, "snap doesn't work in windows")
	testlib.CheckPath(t, "snapcraft")
	folder := t.TempDir()
	dist := filepath.Join(folder, "dist")
	require.NoError(t, os.Mkdir(dist, 0o755))
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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
	}, testctx.WithCurrentTag("v1.2.3"), testctx.WithVersion("1.2.3"))

	addBinaries(t, ctx, "foo", dist)
	require.NoError(t, Pipe{}.Run(ctx))

	apath := filepath.Join(dist, "foo_amd64", "prime", "a", "b", "c", "extra-file.txt")
	bpath := filepath.Join(dist, "foo_amd64", "prime", "testdata", "extra-file-2.txt")
	requireEqualFileContents(t, "testdata/extra-file.txt", apath)
	requireEqualFileContents(t, "testdata/extra-file-2.txt", bpath)
}

func requireEqualFileContents(tb testing.TB, a, b string) {
	tb.Helper()
	eq, err := gio.EqualFileContents(a, b)
	require.NoError(tb, err)
	require.True(tb, eq, "%s != %s", a, b)
}

func addBinaries(t *testing.T, ctx *context.Context, name, dist string) {
	t.Helper()
	for _, goos := range []string{"linux", "darwin"} {
		for _, goarch := range []string{"amd64", "386", "arm"} {
			binPath := filepath.Join(dist, name)
			require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0o755))
			f, err := os.Create(binPath)
			require.NoError(t, err)
			require.NoError(t, f.Close())
			switch goarch {
			case "arm":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "subdir/" + name,
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Goarm:  "6",
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})

			case "amd64":
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:    "subdir/" + name,
					Path:    binPath,
					Goarch:  goarch,
					Goos:    goos,
					Goamd64: "v1",
					Type:    artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})
			default:
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "subdir/" + name,
					Path:   binPath,
					Goarch: goarch,
					Goos:   goos,
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: name,
					},
				})
			}
		}
	}
}
