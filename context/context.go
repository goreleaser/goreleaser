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
	Name, Path, BuildName string
}

// Context carries along some data through the pipes
type Context struct {
	ctx.Context
	Config       config.Project
	Env          map[string]string
	Token        string
	Git          GitInfo
	Binaries     map[string]map[string][]Binary
	Artifacts    []Artifact
	Dockers      []string
	Brews        []string
	ReleaseNotes string
	Version      string
	Validate     bool
	Publish      bool
	Snapshot     bool
	RmDist       bool
	Debug        bool
	Parallelism  int
}

var (
	artifactsLock sync.Mutex
	dockersLock   sync.Mutex
	binariesLock  sync.Mutex
	brewsLock     sync.Mutex
)

type Artifact struct {
	Name, Path string
}

// AddArtifact adds a file to upload list
func (ctx *Context) AddArtifact(name, file string) {
	artifactsLock.Lock()
	defer artifactsLock.Unlock()
	file = strings.TrimPrefix(file, ctx.Config.Dist+string(filepath.Separator))
	ctx.Artifacts = append(ctx.Artifacts, Artifact{Name: name, Path: file})
	log.WithFields(log.Fields{"file": file, "name": name}).Info("new release artifact")
}

func (ctx *Context) AddArtifactFromFile(file string) {
	_, name := filepath.Split(file)
	ctx.AddArtifact(name, file)
}

// AddBrew adds a brew tap to the brews list
func (ctx *Context) AddBrew(tap string) {
	brewsLock.Lock()
	defer brewsLock.Unlock()
	ctx.Brews = append(ctx.Brews, tap)
	log.WithField("tap", tap).Info("new brew tap")
}

// AddDocker adds a docker image to the docker images list
func (ctx *Context) AddDocker(image string) {
	dockersLock.Lock()
	defer dockersLock.Unlock()
	ctx.Dockers = append(ctx.Dockers, image)
	log.WithField("image", image).Info("new docker image")
}

// AddBinary adds a built binary to the current context
func (ctx *Context) AddBinary(platform, folder, name, buildName, path string) {
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
			Name:      name,
			Path:      path,
			BuildName: buildName,
		},
	)
	log.WithField("platform", platform).
		WithField("folder", folder).
		WithField("name", name).
		WithField("path", path).
		WithField("buildName", buildName).
		Debug("new binary")
}

// New context
func New(config config.Project) *Context {
	return &Context{
		Context:     ctx.Background(),
		Config:      config,
		Env:         splitEnv(os.Environ()),
		Parallelism: 4,
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
