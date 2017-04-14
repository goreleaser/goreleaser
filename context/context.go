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
	Diff        string
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
	Version   string
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
