// Package context provides gorelease context which is passed through the
// pipeline.
//
// The context extends the standard library context and add a few more
// fields and other things, so pipes can gather data provided by previous
// pipes without really knowing each other.
package context

import (
	ctx "context"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
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
	Token        string
	Git          GitInfo
	Folders      map[string]string
	Artifacts    []string
	ReleaseNotes string
	Version      string
	Validate     bool
	Publish      bool
	Snapshot     bool
}

var artifactsLock sync.Mutex
var foldersLock sync.Mutex

// AddArtifact adds a file to upload list
func (ctx *Context) AddArtifact(file string) {
	artifactsLock.Lock()
	defer artifactsLock.Unlock()
	file = strings.TrimPrefix(file, ctx.Config.Dist+"/")
	ctx.Artifacts = append(ctx.Artifacts, file)
	log.WithField("artifact", file).Info("new artifact")
}

// AddFolder adds a built binary to the current context
func (ctx *Context) AddFolder(key, folder string) {
	foldersLock.Lock()
	defer foldersLock.Unlock()
	ctx.Folders[key] = folder
	log.WithField("key", key).WithField("folder", folder).Info("new folder")
}

// New context
func New(config config.Project) *Context {
	return &Context{
		Context: ctx.Background(),
		Config:  config,
		Folders: map[string]string{},
	}
}
