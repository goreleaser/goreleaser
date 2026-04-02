package healthcheck

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"

	// langs to init.
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/bun"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/deno"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/golang"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/rust"
	_ "github.com/goreleaser/goreleaser/v2/internal/builders/zig"
)

func TestSystemDependencies(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
	deps := system{}.Dependencies(ctx)
	var names []string
	for _, dep := range deps {
		name, _ := dep()
		names = append(names, name)
	}
	require.Equal(t, []string{"git"}, names)
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
		},
	})
	deps := builds{}.Dependencies(ctx)
	var names []string
	for _, dep := range deps {
		name, _ := dep()
		names = append(names, name)
	}
	require.ElementsMatch(t, []string{
		"bun",
		"deno",
		"go",
		"cargo",
		"rustup",
		"cargo-zigbuild",
		"zig",
		"zig",
	}, names)
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
