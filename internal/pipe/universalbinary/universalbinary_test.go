package universalbinary

import (
	"debug/macho"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestDefault(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "proj",
			UniversalBinaries: []config.UniversalBinary{
				{},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.UniversalBinary{
			ID:           "proj",
			IDs:          []string{"proj"},
			NameTemplate: "{{ .ProjectName }}",
		}, ctx.Config.UniversalBinaries[0])
	})

	t.Run("given ids", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "proj",
			UniversalBinaries: []config.UniversalBinary{
				{IDs: []string{"foo"}},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.UniversalBinary{
			ID:           "proj",
			IDs:          []string{"foo"},
			NameTemplate: "{{ .ProjectName }}",
		}, ctx.Config.UniversalBinaries[0])
	})

	t.Run("given id", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "proj",
			UniversalBinaries: []config.UniversalBinary{
				{ID: "foo"},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.UniversalBinary{
			ID:           "foo",
			IDs:          []string{"foo"},
			NameTemplate: "{{ .ProjectName }}",
		}, ctx.Config.UniversalBinaries[0])
	})

	t.Run("given name", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "proj",
			UniversalBinaries: []config.UniversalBinary{
				{NameTemplate: "foo"},
			},
		})
		require.NoError(t, Pipe{}.Default(ctx))
		require.Equal(t, config.UniversalBinary{
			ID:           "proj",
			IDs:          []string{"proj"},
			NameTemplate: "foo",
		}, ctx.Config.UniversalBinaries[0])
	})

	t.Run("duplicated ids", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			ProjectName: "proj",
			UniversalBinaries: []config.UniversalBinary{
				{ID: "foo"},
				{ID: "foo"},
			},
		})
		require.EqualError(t, Pipe{}.Default(ctx), `found 2 universal_binaries with the ID 'foo', please fix your config`)
	})
}

func TestSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, Pipe{}.Skip(testctx.New()))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			UniversalBinaries: []config.UniversalBinary{{}},
		})
		require.False(t, Pipe{}.Skip(ctx))
	})
}

func TestRun(t *testing.T) {
	dist := t.TempDir()

	src := filepath.Join("testdata", "fake", "main.go")
	paths := map[string]string{
		"amd64": filepath.Join(dist, "fake_darwin_amd64/fake"),
		"arm64": filepath.Join(dist, "fake_darwin_arm64/fake"),
	}

	pre := filepath.Join(dist, "pre")
	post := filepath.Join(dist, "post")
	cfg := config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foo",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
				Replace:      true,
			},
		},
	}
	ctx1 := testctx.NewWithCfg(cfg)

	ctx2 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foo",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
			},
		},
	})

	ctx3 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "notfoo",
				IDs:          []string{"notfoo", "notbar"},
				NameTemplate: "notfoo",
			},
		},
	})

	ctx4 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foo",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
			},
		},
	})

	ctx5 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foo",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
				Hooks: config.BuildHookConfig{
					Pre: []config.Hook{
						{Cmd: testlib.Touch(pre)},
					},
					Post: []config.Hook{
						{Cmd: testlib.Touch(post)},
						{Cmd: testlib.ShC(`echo "{{ .Name }} {{ .Os }} {{ .Arch }} {{ .Arm }} {{ .Target }} {{ .Ext }}" > {{ .Path }}.post`), Output: true},
					},
				},
			},
		},
	})

	ctx6 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foobar",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
			},
		},
	})

	modTime := time.Now().AddDate(-1, 0, 0).Round(1 * time.Second).UTC()
	ctx7 := testctx.NewWithCfg(config.Project{
		Dist: dist,
		UniversalBinaries: []config.UniversalBinary{
			{
				ID:           "foo",
				IDs:          []string{"foo"},
				NameTemplate: "foo",
				ModTimestamp: fmt.Sprintf("%d", modTime.Unix()),
				Hooks: config.BuildHookConfig{
					Pre: []config.Hook{
						{Cmd: testlib.Touch(pre)},
					},
					Post: []config.Hook{
						{Cmd: testlib.Touch(post)},
						{
							Cmd:    testlib.ShC(`echo "{{ .Name }} {{ .Os }} {{ .Arch }} {{ .Arm }} {{ .Target }} {{ .Ext }}" > {{ .Path }}.post`),
							Output: true,
						},
					},
				},
			},
		},
	})

	for arch, path := range paths {
		cmd := exec.Command("go", "build", "-o", path, src)
		cmd.Env = append(os.Environ(), "GOOS=darwin", "GOARCH="+arch)
		_, err := cmd.CombinedOutput()
		require.NoError(t, err)

		modTime := time.Unix(0, 0)
		require.NoError(t, os.Chtimes(path, modTime, modTime))

		art := artifact.Artifact{
			Name:   "fake",
			Path:   path,
			Goos:   "darwin",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				artifact.ExtraBinary: "fake",
				artifact.ExtraID:     "foo",
			},
		}
		ctx1.Artifacts.Add(&art)
		ctx2.Artifacts.Add(&art)
		ctx5.Artifacts.Add(&art)
		ctx6.Artifacts.Add(&art)
		ctx7.Artifacts.Add(&art)
		ctx4.Artifacts.Add(&artifact.Artifact{
			Name:   "fake",
			Path:   path + "wrong",
			Goos:   "darwin",
			Goarch: arch,
			Type:   artifact.Binary,
			Extra: map[string]interface{}{
				artifact.ExtraBinary: "fake",
				artifact.ExtraID:     "foo",
			},
		})
	}

	t.Run("ensure new artifact id", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx6))
		unis := ctx6.Artifacts.Filter(artifact.ByType(artifact.UniversalBinary)).List()
		require.Len(t, unis, 1)
		checkUniversalBinary(t, unis[0])
		require.Equal(t, "foobar", unis[0].ID())
	})

	t.Run("replacing", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx1))
		require.Empty(t, ctx1.Artifacts.Filter(artifact.ByType(artifact.Binary)).List())
		unis := ctx1.Artifacts.Filter(artifact.ByType(artifact.UniversalBinary)).List()
		require.Len(t, unis, 1)
		checkUniversalBinary(t, unis[0])
		require.True(t, artifact.ExtraOr(*unis[0], artifact.ExtraReplaces, false))
	})

	t.Run("keeping", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx2))
		require.Len(t, ctx2.Artifacts.Filter(artifact.ByType(artifact.Binary)).List(), 2)
		unis := ctx2.Artifacts.Filter(artifact.ByType(artifact.UniversalBinary)).List()
		require.Len(t, unis, 1)
		checkUniversalBinary(t, unis[0])
		require.False(t, artifact.ExtraOr(*unis[0], artifact.ExtraReplaces, true))
	})

	t.Run("bad template", func(t *testing.T) {
		testlib.RequireTemplateError(t, Pipe{}.Run(testctx.NewWithCfg(config.Project{
			UniversalBinaries: []config.UniversalBinary{
				{
					NameTemplate: "{{.Name}",
				},
			},
		})))
	})

	t.Run("no darwin builds", func(t *testing.T) {
		require.EqualError(t, Pipe{}.Run(ctx3), `no darwin binaries found with ids: notfoo, notbar`)
	})

	t.Run("fail to open", func(t *testing.T) {
		require.ErrorIs(t, Pipe{}.Run(ctx4), os.ErrNotExist)
	})

	t.Run("hooks", func(t *testing.T) {
		require.NoError(t, Pipe{}.Run(ctx5))
		require.FileExists(t, pre)
		require.FileExists(t, post)
		post := filepath.Join(dist, "foo_darwin_all/foo.post")
		require.FileExists(t, post)
		bts, err := os.ReadFile(post)
		require.NoError(t, err)
		require.Contains(t, string(bts), "foo darwin all  darwin_all")
	})

	t.Run("failing pre-hook", func(t *testing.T) {
		ctx := ctx5
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{Cmd: "exit 1"}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{{Cmd: "doesnt-matter"}}
		err := Pipe{}.Run(ctx)
		require.ErrorIs(t, err, exec.ErrNotFound)
		require.ErrorContains(t, err, "pre hook failed")
	})

	t.Run("failing post-hook", func(t *testing.T) {
		ctx := ctx5
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{Cmd: testlib.Echo("pre")}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{{Cmd: "exit 1"}}
		err := Pipe{}.Run(ctx)
		require.ErrorIs(t, err, exec.ErrNotFound)
		require.ErrorContains(t, err, "post hook failed")
	})

	t.Run("skipping post-hook", func(t *testing.T) {
		ctx := ctx5
		skips.Set(ctx, skips.PostBuildHooks)
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{{Cmd: "exit 1"}}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("skipping pre-hook", func(t *testing.T) {
		ctx := ctx5
		skips.Set(ctx, skips.PreBuildHooks)
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{Cmd: "exit 1"}}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("hook with env tmpl", func(t *testing.T) {
		ctx := ctx5
		ctx.Skips[string(skips.PostBuildHooks)] = false
		ctx.Skips[string(skips.PreBuildHooks)] = false
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{
			Cmd: testlib.Echo("{{.Env.FOO}}"),
			Env: []string{"FOO=foo-{{.Tag}}"},
		}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{}
		require.NoError(t, Pipe{}.Run(ctx))
	})

	t.Run("hook with bad env tmpl", func(t *testing.T) {
		ctx := ctx5
		ctx.Skips[string(skips.PostBuildHooks)] = false
		ctx.Skips[string(skips.PreBuildHooks)] = false
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{
			Cmd: testlib.Echo("blah"),
			Env: []string{"FOO=foo-{{.Tag}"},
		}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{}
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("hook with bad dir tmpl", func(t *testing.T) {
		ctx := ctx5
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{
			Cmd: testlib.Echo("blah"),
			Dir: "{{.Tag}",
		}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{}
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("hook with bad cmd tmpl", func(t *testing.T) {
		ctx := ctx5
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{{
			Cmd: testlib.Echo("blah-{{.Tag }"),
		}}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{}
		testlib.RequireTemplateError(t, Pipe{}.Run(ctx))
	})

	t.Run("mod timestamp", func(t *testing.T) {
		ctx := ctx7
		require.NoError(t, Pipe{}.Run(ctx))
		unibins := ctx.Artifacts.Filter(artifact.ByType(artifact.UniversalBinary)).List()
		require.Len(t, unibins, 1)
		stat, err := os.Stat(unibins[0].Path)
		require.NoError(t, err)
		require.Equal(t, modTime.Unix(), stat.ModTime().Unix())
	})

	t.Run("bad mod timestamp", func(t *testing.T) {
		ctx := ctx5
		ctx.Config.UniversalBinaries[0].ModTimestamp = "not a number"
		ctx.Config.UniversalBinaries[0].Hooks.Pre = []config.Hook{}
		ctx.Config.UniversalBinaries[0].Hooks.Post = []config.Hook{}
		require.ErrorIs(t, Pipe{}.Run(ctx), strconv.ErrSyntax)
	})
}

func checkUniversalBinary(tb testing.TB, unibin *artifact.Artifact) {
	tb.Helper()

	require.True(tb, strings.HasSuffix(unibin.Path, unibin.ID()+"_darwin_all/foo"))
	f, err := macho.OpenFat(unibin.Path)
	require.NoError(tb, err)
	require.Len(tb, f.Arches, 2)
}
