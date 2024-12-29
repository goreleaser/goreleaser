package build

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testctx"
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

// type constraints
var (
	_ Builder           = &dummy{}
	_ Builder           = &completeDummy{}
	_ PreparedBuilder   = &completeDummy{}
	_ ConcurrentBuilder = &completeDummy{}
	_ DependingBuilder  = &completeDummy{}
)

type dummy struct{}

func (*dummy) Parse(string) (Target, error)                              { return dummyTarget{}, nil }
func (*dummy) WithDefaults(build config.Build) (config.Build, error)     { return build, nil }
func (*dummy) Build(_ *context.Context, _ config.Build, _ Options) error { return nil }

type completeDummy struct{}

func (*completeDummy) Dependencies() []string                                    { return []string{"fake"} }
func (*completeDummy) AllowConcurrentBuilds() bool                               { return true }
func (*completeDummy) Prepare(*context.Context, config.Build) error              { return nil }
func (*completeDummy) Parse(string) (Target, error)                              { return dummyTarget{}, nil }
func (*completeDummy) WithDefaults(build config.Build) (config.Build, error)     { return build, nil }
func (*completeDummy) Build(_ *context.Context, _ config.Build, _ Options) error { return nil }

var (
	defaultCompleteDummy = &completeDummy{}
	defaultDummy         = &dummy{}
)

func TestMain(m *testing.M) {
	Register("completedummy", defaultCompleteDummy)
	Register("dummy", defaultDummy)
	m.Run()
}

func TestRegisterAndGet(t *testing.T) {
	require.Equal(t, defaultDummy, For("dummy"))
	require.Equal(t, defaultCompleteDummy, For("completedummy"))
}

func TestDependencies(t *testing.T) {
	require.Equal(t, []string{"fake"}, Dependencies(testctx.NewWithCfg(
		config.Project{
			Builds: []config.Build{
				{Builder: "completedummy"},
				{Builder: "dummy"},
			},
		},
	)))
}
