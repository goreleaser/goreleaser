package healthcheck

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"

	// langs to init.
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/bun"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/deno"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/golang"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/node"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/rust"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/zig"
)

func TestSystemDependencies(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
	require.Equal(t, []string{"git"}, system{}.Dependencies(ctx))
}

func TestSystemStringer(t *testing.T) {
	require.NotEmpty(t, system{}.String())
}

func TestBuildDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{Builder: "bun"},
			{Builder: "deno"},
			{Builder: "go"},
			{Builder: "rust"},
			{Builder: "zig"},
			{Builder: "node"},
		},
	})
	require.Equal(t, []string{
		"bun",
		"deno",
		"go",
		"cargo",
		"rustup",
		"cargo-zigbuild",
		"zig",
		"zig", // dedup happens later on
		"node",
	}, builds{}.Dependencies(ctx))
}

func TestBuildStringer(t *testing.T) {
	require.NotEmpty(t, builds{}.String())
}

func TestHealthCheckers(t *testing.T) {
	require.NotEmpty(t, HealthCheckers)
}

func TestDependencyCheckers(t *testing.T) {
	require.NotEmpty(t, DependencyCheckers)
}

// Pipes that document requiring an external binary in $PATH must be part of
// the dependency checkers, otherwise `goreleaser healthcheck` reports a clean
// bill of health for a config that cannot run.
func TestDependencyCheckersCoverExternalBinaries(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Makeselfs: []config.Makeself{{Script: "install.sh"}},
		UPXs:      []config.UPX{{Enabled: "true"}},
	})
	var tools []string
	for _, hc := range DependencyCheckers {
		if skipper, ok := hc.(interface {
			Skip(*context.Context) bool
		}); ok && skipper.Skip(ctx) {
			continue
		}
		tools = append(tools, hc.Dependencies(ctx)...)
	}
	require.Contains(t, tools, "makeself")
	require.Contains(t, tools, "upx")
}
