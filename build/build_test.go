package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, builder, For("dummy"))
}
