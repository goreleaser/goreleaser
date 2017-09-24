// Package context provides gorelease context which is passed through the
// pipeline.
//
// The context extends the standard library context and add a few more
// fields and other things, so pipes can gather data provided by previous
// pipes without really knowing each other.
package context

import (
	ctx "context"
	"path/filepath"
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

// Binary with pretty name and path
type Binary struct {
	Name, Path string
}

// Context carries along some data through the pipes
type Context struct {
	ctx.Context
	Config       config.Project
	Token        string
	Git          GitInfo
	Binaries     map[string]map[string][]Binary
	Artifacts    []string
	Dockers      []string
	ReleaseNotes string
	Version      string
	Validate     bool
	Publish      bool
	Snapshot     bool
	RmDist       bool
	Parallelism  int
}

var artifactsLock sync.Mutex
var dockersLock sync.Mutex
var binariesLock sync.Mutex

// AddArtifact adds a file to upload list
func (ctx *Context) AddArtifact(file string) {
	artifactsLock.Lock()
	defer artifactsLock.Unlock()
	file = strings.TrimPrefix(file, ctx.Config.Dist+string(filepath.Separator))
	ctx.Artifacts = append(ctx.Artifacts, file)
	log.WithField("artifact", file).Info("new release artifact")
}

// AddDocker adds a docker image to the docker images list
func (ctx *Context) AddDocker(image string) {
	dockersLock.Lock()
	defer dockersLock.Unlock()
	ctx.Dockers = append(ctx.Dockers, image)
	log.WithField("image", image).Info("new docker image")
}

// AddBinary adds a built binary to the current context
func (ctx *Context) AddBinary(platform, folder, name, path string) {
	binariesLock.Lock()
	defer binariesLock.Unlock()
	if ctx.Binaries == nil {
		ctx.Binaries = map[string]map[string][]Binary{}
	}
	if ctx.Binaries[platform] == nil {
		ctx.Binaries[platform] = map[string][]Binary{}
	}
	ctx.Binaries[platform][folder] = append(
		ctx.Binaries[platform][folder],
		Binary{
			Name: name,
			Path: path,
		},
	)
	log.WithField("platform", platform).
		WithField("folder", folder).
		WithField("name", name).
		WithField("path", path).
		Debug("new binary")
}

// New context
func New(config config.Project) *Context {
	return &Context{
		Context:     ctx.Background(),
		Config:      config,
		Parallelism: 4,
	}
}
