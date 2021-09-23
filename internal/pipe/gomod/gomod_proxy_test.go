package gomod

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestGoModProxy(t *testing.T) {
	t.Run("goreleaser", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
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
		})
		ctx.Git.CurrentTag = "v0.161.1"

		ctx.ModulePath = "github.com/goreleaser/goreleaser"

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t, ctx.ModulePath, ctx.Git.CurrentTag)
		requireMainGo(t, ctx.ModulePath)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, ".", ctx.Config.Builds[0].UnproxiedMain)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, ".", ctx.Config.Builds[0].UnproxiedDir)
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
	})

	t.Run("nfpm", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
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
		})
		ctx.Git.CurrentTag = "v2.3.1"

		ctx.ModulePath = "github.com/goreleaser/nfpm/v2"
		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t, ctx.ModulePath, ctx.Git.CurrentTag)
		requireMainGo(t, ctx.ModulePath+"/cmd/nfpm")
		require.Equal(t, ctx.ModulePath+"/cmd/nfpm", ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
	})

	// this repo does not have a go.sum file, which is ok, a project might not have any dependencies
	t.Run("no go.sum", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
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
		})
		ctx.Git.CurrentTag = "v0.0.1"

		ctx.ModulePath = "github.com/goreleaser/example-mod-proxy"
		fakeGoMod(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t, ctx.ModulePath, ctx.Git.CurrentTag)
		requireMainGo(t, ctx.ModulePath)
		require.Equal(t, ctx.ModulePath, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, ctx.ModulePath, ctx.ModulePath)
	})

	t.Run("no perms", func(t *testing.T) {
		for file, mode := range map[string]os.FileMode{
			"go.mod":          0o500,
			"go.sum":          0o500,
			"main.go":         0o500,
			"../../../go.sum": 0o300,
		} {
			t.Run(file, func(t *testing.T) {
				dir := testlib.Mktmp(t)
				dist := filepath.Join(dir, "dist")
				ctx := context.New(config.Project{
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
				})
				ctx.Git.CurrentTag = "v0.161.1"

				ctx.ModulePath = "github.com/goreleaser/goreleaser"

				fakeGoModAndSum(t, ctx.ModulePath)
				require.NoError(t, ProxyPipe{}.Run(ctx)) // should succeed at first

				// change perms of a file and run again, which should now fail on that file.
				require.NoError(t, os.Chmod(filepath.Join(dist, "proxy", "foo", file), mode))
				require.ErrorAs(t, ProxyPipe{}.Run(ctx), &ErrProxy{})
			})
		}
	})

	t.Run("goreleaser with main.go", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
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
		})
		ctx.Git.CurrentTag = "v0.161.1"

		ctx.ModulePath = "github.com/goreleaser/goreleaser"

		fakeGoModAndSum(t, ctx.ModulePath)
		require.NoError(t, ProxyPipe{}.Run(ctx))
		requireGoMod(t, ctx.ModulePath, ctx.Git.CurrentTag)
		requireMainGo(t, ctx.ModulePath)
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
		require.True(t, ProxyPipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("skip snapshot", func(t *testing.T) {
		ctx := context.New(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		})
		ctx.ModulePath = "github.com/goreleaser/goreleaser"
		ctx.Snapshot = true
		require.True(t, ProxyPipe{}.Skip(ctx))
	})

	t.Run("skip not a go module", func(t *testing.T) {
		ctx := context.New(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		})
		ctx.ModulePath = ""
		require.True(t, ProxyPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			GoMod: config.GoMod{
				Proxy: true,
			},
		})
		ctx.ModulePath = "github.com/goreleaser/goreleaser"
		require.False(t, ProxyPipe{}.Skip(ctx))
	})
}

func requireGoMod(tb testing.TB, module, version string) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Contains(tb, string(mod), fmt.Sprintf(`module foo

go 1.17

require %s %s
`, module, version))
}

func requireMainGo(tb testing.TB, module string) {
	tb.Helper()

	main, err := os.ReadFile("dist/proxy/foo/main.go")
	require.NoError(tb, err)
	require.Equal(tb, fmt.Sprintf(`
// +build main
package main

import _ "%s"
`, module), string(main))
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
