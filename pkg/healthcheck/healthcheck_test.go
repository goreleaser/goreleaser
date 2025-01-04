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
	ctx := testctx.New()
	require.Equal(t, []string{"git"}, system{}.Dependencies(ctx))
}

func TestSystemStringer(t *testing.T) {
	require.NotEmpty(t, system{}.String())
}

func TestBuildDependencies(t *testing.T) {
	ctx := testctx.NewWithCfg(config.Project{
		Builds: []config.Build{
			{Builder: "bun"},
			{Builder: "deno"},
			{Builder: "go"},
			{Builder: "rust"},
			{Builder: "zig"},
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
	}, builds{}.Dependencies(ctx))
}

func TestBuildStringer(t *testing.T) {
	require.NotEmpty(t, builds{}.String())
}
