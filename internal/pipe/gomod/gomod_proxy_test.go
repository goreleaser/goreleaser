package gomod

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testctx"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

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

		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
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
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
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
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
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
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
	})
}

func TestProxyDescription(t *testing.T) {
	require.NotEmpty(t, ProxyPipe{}.String())
}

func TestSkip(t *testing.T) {
	t.Run("skip false gomod.proxy", func(t *testing.T) {
		require.True(t, ProxyPipe{}.Skip(testctx.New()))
	})

	t.Run("skip snapshot", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, withGoReleaserModulePath, testctx.Snapshot)
		require.True(t, ProxyPipe{}.Skip(ctx))
	})

	t.Run("skip not a go module", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, func(ctx *context.Context) { ctx.ModulePath = "" })
		require.True(t, ProxyPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := testctx.NewWithCfg(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		}, withGoReleaserModulePath)
		require.False(t, ProxyPipe{}.Skip(ctx))
	})
}

func requireGoMod(tb testing.TB) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Contains(tb, string(mod), `module foo

go 1.21
`)
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
