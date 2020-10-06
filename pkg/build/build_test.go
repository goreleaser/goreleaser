package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

type dummy struct{}

func (*dummy) WithDefaults(build config.Build) config.Build {
	return build
}
func (*dummy) Build(ctx *context.Context, build config.Build, options Options) error {
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	var builder = &dummy{}
	Register("dummy", builder)
	require.Equal(t, builder, For("dummy"))
}
