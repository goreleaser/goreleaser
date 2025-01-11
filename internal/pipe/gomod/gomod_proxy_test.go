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
		}, testctx.Snapshot, withTestModulePath)

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
		}, withTestModulePath)

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
		}, withTestModulePath)

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, exec.Command("go", "mod", "edit", "-replace", "foo=../bar").Run())
		require.ErrorIs(t, CheckGoModPipe{}.Run(ctx), ErrReplaceWithProxy)
	})
}

func TestGoModProxy(t *testing.T) {
	t.Run("testmod", func(t *testing.T) {
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
					Main:   "./cmd/fake",
				},
			},
		}, testctx.WithCurrentTag("v0.1.1"), func(ctx *context.Context) {
			ctx.ModulePath = "github.com/goreleaser/test-mod"
		})
		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath+"/cmd/fake", ctx.Config.Builds[0].Main)
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
		}, testctx.WithCurrentTag("v0.0.1"), func(ctx *context.Context) {
			ctx.ModulePath = "github.com/goreleaser/example-mod-proxy"
		})
		fakeGoMod(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
	})

	t.Run("no perms", func(t *testing.T) {
		testlib.SkipIfWindows(t, "windows permissions work differently")
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
				}, withTestModulePath, testctx.WithCurrentTag("v0.1.1"))

				fakeGoModAndSum(t, ctx.ModulePath)
				require.NoError(t, ProxyPipe{}.Run(ctx)) // should succeed at first

				// change perms of a file and run again, which should now fail on that file.
				require.NoError(t, os.Chmod(filepath.Join(dist, "proxy", "foo", file), mode))
				err := ProxyPipe{}.Run(ctx)
				require.Error(t, err)
				if !testlib.IsWindows() {
					// this fails on windows
					require.ErrorIs(t, err, syscall.EACCES)
				}
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
		}, withTestModulePath, testctx.WithCurrentTag("v0.1.1"))

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

func TestSkipProxy(t *testing.T) {
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
		}, withTestModulePath, testctx.Snapshot)
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
		}, withTestModulePath)
		require.False(t, ProxyPipe{}.Skip(ctx))
		require.False(t, CheckGoModPipe{}.Skip(ctx))
	})
}

func requireGoMod(tb testing.TB) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Contains(tb, string(mod), `module foo

go 1.23`)
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

func withTestModulePath(ctx *context.Context) {
	ctx.ModulePath = "github.com/goreleaser/test-mod"
}

func withExampleModulePath(ctx *context.Context) {
	ctx.ModulePath = "github.com/goreleaser/example-mod-proxy"
}
