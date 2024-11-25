package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

type dummyTarget struct{}

// String implements Target.
func (d dummyTarget) String() string {
	return "dummy"
}

// Fields implements Target.
func (d dummyTarget) Fields() map[string]string {
	return nil
}

type dummy struct{}

// Parse implements Builder.
func (d *dummy) Parse(string) (Target, error) {
	return dummyTarget{}, nil
}

func (*dummy) WithDefaults(build config.Build) (config.Build, error) {
	return build, nil
}

func (*dummy) Build(_ *context.Context, _ config.Build, _ Options) error {
	return nil
}

func TestRegisterAndGet(t *testing.T) {
	builder := &dummy{}
	Register("dummy", builder)
	require.Equal(t, builder, For("dummy"))
}
