package gomod

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
		ctx.ModulePath = "github.com/goreleaser/goreleaser"

		setupGoModSumFromURL(t, "https://raw.githubusercontent.com/goreleaser/goreleaser/v0.161.1")
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		requireGoMod(t, "github.com/goreleaser/goreleaser", "v0.161.1")
		requireMainGo(t, "github.com/goreleaser/goreleaser")
		require.Equal(t, "github.com/goreleaser/goreleaser", ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
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
		ctx.ModulePath = "github.com/goreleaser/nfpm/v2"

		setupGoModSumFromURL(t, "https://raw.githubusercontent.com/goreleaser/nfpm/v2.3.1")
		require.NoError(t, Pipe{}.Default(ctx))
		require.NoError(t, Pipe{}.Run(ctx))
		requireGoMod(t, "github.com/goreleaser/nfpm/v2", "v2.3.1")
		requireMainGo(t, "github.com/goreleaser/nfpm/v2/cmd/nfpm")
		require.Equal(t, "github.com/goreleaser/nfpm/v2/cmd/nfpm", ctx.Config.Builds[0].Main)
		require.Equal(t, filepath.Join(dist, "proxy", "foo"), ctx.Config.Builds[0].Dir)
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

func setupGoModSumFromURL(tb testing.TB, url string) {
	tb.Helper()

	require.NoError(tb, os.WriteFile("go.mod", getBody(tb, url+"/go.mod"), 0o666))
	require.NoError(tb, os.WriteFile("go.sum", getBody(tb, url+"/go.sum"), 0o666))
}

func getBody(tb testing.TB, url string) []byte {
	tb.Log("URL", url)
	res, err := http.Get(url)
	require.NoError(tb, err)
	tb.Cleanup(func() {
		require.NoError(tb, res.Body.Close())
	})
	bts, err := ioutil.ReadAll(res.Body)
	require.NoError(tb, err)
	return bts
}
