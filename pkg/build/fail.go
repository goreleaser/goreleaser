package build

import (
	"errors"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

var (
	_ Builder           = failBuilder{}
	_ DependingBuilder  = failBuilder{}
	_ PreparedBuilder   = failBuilder{}
	_ ConcurrentBuilder = failBuilder{}
	_ TargetFixer       = failBuilder{}
)

func newFail(name string) failBuilder {
	return failBuilder{
		err: errors.New("invalid builder: " + name),
	}
}

type failBuilder struct {
	err error
}

func (f failBuilder) WithDefaults(b config.Build) (config.Build, error)   { return b, f.err }
func (f failBuilder) Build(*context.Context, config.Build, Options) error { return f.err }
func (f failBuilder) Parse(string) (Target, error)                        { return nil, f.err }
func (f failBuilder) Dependencies() []string                              { return nil }
func (f failBuilder) Prepare(*context.Context, config.Build) error        { return f.err }
func (f failBuilder) AllowConcurrentBuilds() bool                         { return false }
func (f failBuilder) FixTarget(string) string                             { return "" }
