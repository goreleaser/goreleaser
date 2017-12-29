// Package context provides gorelease context which is passed through the
// pipeline.
//
// The context extends the standard library context and add a few more
// fields and other things, so pipes can gather data provided by previous
// pipes without really knowing each other.
package context

import (
	ctx "context"
	"os"
	"strings"
	"time"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/internal/artifact"
)

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag string
	Commit     string
}

// Context carries along some data through the pipes
type Context struct {
	ctx.Context
	Config       config.Project
	Env          map[string]string
	Token        string
	Git          GitInfo
	Artifacts    artifact.Artifacts
	ReleaseNotes string
	Version      string
	Validate     bool
	Publish      bool
	Snapshot     bool
	RmDist       bool
	Debug        bool
	Parallelism  int
}

// New context
func New(config config.Project) *Context {
	return wrap(ctx.Background(), config)
}

// NewWithTimeout new context with the given timeout
func NewWithTimeout(config config.Project, timeout time.Duration) (*Context, ctx.CancelFunc) {
	ctx, cancel := ctx.WithTimeout(ctx.Background(), timeout)
	return wrap(ctx, config), cancel
}

func wrap(ctx ctx.Context, config config.Project) *Context {
	return &Context{
		Context:     ctx,
		Config:      config,
		Env:         splitEnv(os.Environ()),
		Parallelism: 4,
		Artifacts:   artifact.New(),
	}
}

func splitEnv(env []string) map[string]string {
	r := map[string]string{}
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		r[p[0]] = p[1]
	}
	return r
}
