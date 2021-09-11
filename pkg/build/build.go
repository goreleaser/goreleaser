// Package build provides the API for external builders
package build

import (
	"sync"

	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// nolint: gochecknoglobals
var (
	builders = map[string]Builder{}
	lock     sync.Mutex
)

// Register registers a builder to a given name.
func Register(name string, builder Builder) {
	lock.Lock()
	builders[name] = builder
	lock.Unlock()
}

// For gets the previously registered builder for the given name.
func For(name string) Builder {
	return builders[name]
}

// Options to be passed down to a builder.
type Options struct {
	Name   string
	Path   string
	Ext    string
	Target string
	Goos   string
	Goarch string
	Goarm  string
	Gomips string
}

// Builder defines a builder.
type Builder interface {
	WithDefaults(build config.Build) (config.Build, error)
	Build(ctx *context.Context, build config.Build, options Options) error
}
