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

func TestRun(t *testing.T) {
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser", ctx.ModulePath)
}

func TestRunSnapshot(t *testing.T) {
	ctx := context.New(config.Project{
		GoMod: config.GoMod{
			Proxy: true,
		},
	})
	ctx.Snapshot = true
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Equal(t, "github.com/goreleaser/goreleaser", ctx.ModulePath)
}

func TestRunOutsideGoModule(t *testing.T) {
	dir := testlib.Mktmp(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {println(0)}"), 0o666))
	ctx := context.New(config.Project{})
	require.NoError(t, Pipe{}.Default(ctx))
	testlib.AssertSkipped(t, Pipe{}.Run(ctx))
	require.Empty(t, ctx.ModulePath)
}

func TestRunCommandError(t *testing.T) {
	ctx := context.New(config.Project{
		GoMod: config.GoMod{
			GoBinary: "not-a-valid-binary",
		},
	})
	require.EqualError(t, Pipe{}.Run(ctx), "failed to get module path: exec: \"not-a-valid-binary\": executable file not found in $PATH: ")
	require.Empty(t, ctx.ModulePath)
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestGoModProxy(t *testing.T) {
	t.Run("goreleaser", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy: true,
			},
			Builds: []config.Build{
				{
					ID:     "foo",
					Goos:   []string{runtime.GOOS},
					Goarch: []string{runtime.GOARCH},
					Main:   ".",
				},
			},
		})
		ctx.Git.CurrentTag = "v0.161.1"

		mod := "github.com/goreleaser/goreleaser"

		fakeGoModAndSum(t, mod)
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		requireGoMod(t, mod, ctx.Git.CurrentTag)
		requireMainGo(t, mod)
		require.Equal(t, mod, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, mod, ctx.ModulePath)
	})

	t.Run("nfpm", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy: true,
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

		mod := "github.com/goreleaser/nfpm/v2"
		fakeGoModAndSum(t, mod)
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		requireGoMod(t, mod, ctx.Git.CurrentTag)
		requireMainGo(t, mod+"/cmd/nfpm")
		require.Equal(t, mod+"/cmd/nfpm", ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, mod, ctx.ModulePath)
	})

	// this repo does not have a go.sum file, which is ok, a project might not have any dependencies
	t.Run("no go.sum", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := context.New(config.Project{
			Dist: dist,
			GoMod: config.GoMod{
				Proxy: true,
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

		mod := "github.com/goreleaser/example-mod-proxy"
		fakeGoMod(t, mod)
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		requireGoMod(t, mod, ctx.Git.CurrentTag)
		requireMainGo(t, mod)
		require.Equal(t, mod, ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
		require.Equal(t, mod, ctx.ModulePath)
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
						Proxy: true,
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

				mod := "github.com/goreleaser/goreleaser"

				fakeGoModAndSum(t, mod)
				require.NoError(t, Pipe{}.Default(ctx))
				require.NoError(t, Pipe{}.Run(ctx)) // should succeed at first

				// change perms of a file and run again, which should now fail on that file.
				require.NoError(t, os.Chmod(filepath.Join(dist, "proxy", "foo", file), mode))
				require.ErrorAs(t, Pipe{}.Run(ctx), &ErrProxy{})
			})
		}
	})
}

func requireGoMod(tb testing.TB, module, version string) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Equal(tb, fmt.Sprintf(`module foo

go 1.16

require %s %s
`, module, version), string(mod))
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
