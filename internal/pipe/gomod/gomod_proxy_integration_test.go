//go:build integration

package gomod

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestIntegrationGoModProxy(t *testing.T) {
	t.Run("testmod", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

	t.Run("no go.sum", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

	t.Run("goreleaser with main.go", func(t *testing.T) {
		dir := testlib.Mktmp(t)
		dist := filepath.Join(dir, "dist")
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
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

func requireGoMod(tb testing.TB) {
	tb.Helper()

	mod, err := os.ReadFile("dist/proxy/foo/go.mod")
	require.NoError(tb, err)
	require.Contains(tb, string(mod), fmt.Sprintf(`module foo

go %s
`, testlib.GoVersion))
}
