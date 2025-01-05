// Package build provides the API for external builders
package build

import (
	"sync"

	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

//nolint:gochecknoglobals
var (
	builders = map[string]Builder{}
	lock     sync.RWMutex
)

// Register registers a builder to a given name.
func Register(name string, builder Builder) {
	lock.Lock()
	defer lock.Unlock()
	builders[name] = builder
}

// For gets the previously registered builder for the given name.
func For(name string) Builder {
	lock.RLock()
	defer lock.RUnlock()
	b, ok := builders[name]
	if !ok {
		return newFail(name)
	}
	return b
}

// Dependencies returns all dependencies from all builders being used.
func Dependencies(ctx *context.Context) []string {
	var result []string
	for _, build := range ctx.Config.Builds {
		dep, ok := For(build.Builder).(DependingBuilder)
		if !ok {
			continue
		}
		result = append(result, dep.Dependencies()...)
	}
	return result
}

// Options to be passed down to a builder.
type Options struct {
	Name   string
	Path   string
	Ext    string // with the leading `.`.
	Target Target
}

// Target represents a build target.
//
// Each Builder implementation can implement its own.
type Target interface {
	// String returns the original target.
	String() string

	// Fields returns the template fields that will be available for this
	// target (e.g. Os, Arch, etc).
	Fields() map[string]string
}

// Builder defines a builder.
type Builder interface {
	WithDefaults(build config.Build) (config.Build, error)
	Build(ctx *context.Context, build config.Build, options Options) error
	Parse(target string) (Target, error)
}

// DependingBuilder can be implemented by builders that have dependencies.
type DependingBuilder interface {
	Dependencies() []string
}

// PreparedBuilder can be implemented to run something before all the actual
// builds happen.
type PreparedBuilder interface {
	Prepare(ctx *context.Context, build config.Build) error
}

// ConcurrentBuilder can be implemented to indicate whether or not this builder
// support concurrent builds.
type ConcurrentBuilder interface {
	AllowConcurrentBuilds() bool
}

// TargetFixer allows the builder to provide a way to "default" an incomplete
// target, e.g., on Go, 'darwin_arm64' would need to be defaulted to
// 'darwin_arm64_v8.0'.
type TargetFixer interface {
	FixTarget(target string) string
}
