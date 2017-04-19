// Package context provides gorelease context which is passed through the
// pipeline.
//
// The context extends the standard library context and add a few more
// fields and other things, so pipes can gather data provided by previous
// pipes without really knowing each other.
package context

import (
	ctx "context"
	"log"
	"strings"
	"sync"

	"github.com/goreleaser/goreleaser/config"
)

// GitInfo includes tags and diffs used in some point
type GitInfo struct {
	CurrentTag  string
	PreviousTag string
	Commit      string
}

// Context carries along some data through the pipes
type Context struct {
	ctx.Context
	Config    config.Project
	Token     string
	Git       GitInfo
	Archives  map[string]string
	Artifacts []string
	Changelog string
	Version   string
	Validate  bool
	Publish   bool
}

var lock sync.Mutex

// AddArtifact adds a file to upload list
func (ctx *Context) AddArtifact(file string) {
	lock.Lock()
	defer lock.Unlock()
	file = strings.TrimPrefix(file, ctx.Config.Dist)
	file = strings.Replace(file, "/", "", -1)
	ctx.Artifacts = append(ctx.Artifacts, file)
	log.Println("Registered artifact", file)
}

// New context
func New(config config.Project) *Context {
	return &Context{
		Context:  ctx.Background(),
		Config:   config,
		Archives: map[string]string{},
	}
}
