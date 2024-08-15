package gomod

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	require.NotEmpty(t, CheckGoModPipe{}.String())
	require.NotEmpty(t, ProxyPipe{}.String())
}

func TestCheckGoMod(t *testing.T) {
	t.Run("replace on snapshot", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   ".",
					Dir:    ".",
				},
			},
		}, testctx.Snapshot, withGoReleaserModulePath)

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, exec.Command("go", "mod", "edit", "-replace", "foo=../bar").Run())
		require.NoError(t, CheckGoModPipe{}.Run(ctx))
	})
	t.Run("no go mod", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   ".",
					Dir:    ".",
				},
			},
		}, withGoReleaserModulePath)

		require.NoError(t, CheckGoModPipe{}.Run(ctx))
	})
	t.Run("replace", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   ".",
					Dir:    ".",
				},
			},
		}, withGoReleaserModulePath)

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, exec.Command("go", "mod", "edit", "-replace", "foo=../bar").Run())
		require.ErrorIs(t, CheckGoModPipe{}.Run(ctx), ErrReplaceWithProxy)
	})
}

func TestGoModProxy(t *testing.T) {
	t.Run("goreleaser", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   ".",
					Dir:    ".",
				},
			},
		}, testctx.WithCurrentTag("v0.161.1"), withGoReleaserModulePath)

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, ".", ctx.Config.Builds[0].UnproxiedMain)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, ".", ctx.Config.Builds[0].UnproxiedDir)
	})

	t.Run("nfpm", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   "./cmd/nfpm",
				},
			},
		}, testctx.WithCurrentTag("v2.3.1"), withNfpmModulePath)
		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath+"/cmd/nfpm", ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
	})

	// this repo does not have a go.sum file, which is ok, a project might not have any dependencies
	t.Run("no go.sum", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
				},
			},
		}, testctx.WithCurrentTag("v0.0.1"), withExampleModulePath)
		fakeGoMod(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
	})

	t.Run("no perms", func(t *testing.T) {
		for file, mode := range map[string]os.FileMode{
			"go.mod":          0o500,
			"go.sum":          0o500,
			"../../../go.sum": 0o300,
		} {
			t.Run(file, func(t *testing.T) {
				dir := testlib.Mktmp(t)
				dist := filepath.Join(dir, "dist")
				ctx := testctx.NewWithCfg(config.Project{
					Dist: dist,
					GoMod: config.GoMod{
						Proxy:    true,
						GoBinary: "go",
					},
					Builds: []config.Build{
						{
							ID:     "foo",
							Goos:   []string{runtime.GOOS},
							Goarch: []string{runtime.GOARCH},
						},
					},
				}, withGoReleaserModulePath, testctx.WithCurrentTag("v0.161.1"))

				fakeGoModAndSum(t, ctx.ModulePath)
				require.NoError(t, ProxyPipe{}.Run(ctx)) // should succeed at first

				// change perms of a file and run again, which should now fail on that file.
				require.NoError(t, os.Chmod(filepath.Join(dist, "proxy", "foo", file), mode))
				require.ErrorIs(t, ProxyPipe{}.Run(ctx), syscall.EACCES)
			})
		}
	})

	t.Run("goreleaser with main.go", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.NewWithCfg(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy:    true,
				GoBinary: "go",
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   "main.go",
				},
			},
		}, withGoReleaserModulePath, testctx.WithCurrentTag("v0.161.1"))

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
	})
}

func TestProxyDescription(t *testing.T) {
	require.NotEmpty(t, ProxyPipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip false gomod.proxy", func(t *testing.T) {
		ctx := testctx.New()
		require.True(t, ProxyPipe{}.Skip(ctx))
		require.True(t, CheckGoModPipe{}.Skip(ctx))
	})

	t.Run("skip snapshot", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, withGoReleaserModulePath, testctx.Snapshot)
		require.True(t, ProxyPipe{}.Skip(ctx))
		require.False(t, CheckGoModPipe{}.Skip(ctx))
	})

	t.Run("skip not a go module", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, func(ctx *context.Context) { ctx.ModulePath = "" })
		require.True(t, ProxyPipe{}.Skip(ctx))
		require.True(t, CheckGoModPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, withGoReleaserModulePath)
		require.False(t, ProxyPipe{}.Skip(ctx))
		require.False(t, CheckGoModPipe{}.Skip(ctx))
	})
}

func requireGoMod(tb testing.TB) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Contains(tb, string(mod), `module foo

go 1.2`)
}

func fakeGoModAndSum(tb testing.TB, module string) {
	tb.Helper()

	fakeGoMod(tb, module)
	require.NoError(tb, os.WriteFile("go.sum", []byte("\n"), 0o666))
}

func fakeGoMod(tb testing.TB, module string) {
	tb.Helper()
	require.NoError(tb, os.WriteFile("go.mod", []byte(fmt.Sprintf("module %s\n", module)), 0o666))
}

func withGoReleaserModulePath(ctx *context.Context) {
	ctx.ModulePath = "github.com/goreleaser/goreleaser"
}

func withNfpmModulePath(ctx *context.Context) {
	ctx.ModulePath = "github.com/goreleaser/nfpm/v2"
}

func withExampleModulePath(ctx *context.Context) {
	ctx.ModulePath = "github.com/goreleaser/example-mod-proxy"
}
