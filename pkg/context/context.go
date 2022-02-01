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
	"runtime"
	"strings"
	"time"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// GitInfo includes tags and diffs used in some point.
type GitInfo struct {
	Branch      string
	CurrentTag  string
	PreviousTag string
	Commit      string
	ShortCommit string
	FullCommit  string
	CommitDate  time.Time
	URL         string
	Summary     string
	TagSubject  string
	TagContents string
}

// Env is the environment variables.
type Env map[string]string

// Copy returns a copy of the environment.
func (e Env) Copy() Env {
	out := Env{}
	for k, v := range e {
		out[k] = v
	}
	return out
}

// Strings returns the current environment as a list of strings, suitable for
// os executions.
func (e Env) Strings() []string {
	result := make([]string, 0, len(e))
	for k, v := range e {
		result = append(result, k+"="+v)
	}
	return result
}

// TokenType is either github or gitlab.
type TokenType string

const (
	// TokenTypeGitHub defines github as type of the token.
	TokenTypeGitHub TokenType = "github"
	// TokenTypeGitLab defines gitlab as type of the token.
	TokenTypeGitLab TokenType = "gitlab"
	// TokenTypeGitea defines gitea as type of the token.
	TokenTypeGitea TokenType = "gitea"
)

// Context carries along some data through the pipes.
type Context struct {
	ctx.Context
	Config             config.Project
	Env                Env
	SkipTokenCheck     bool
	Token              string
	TokenType          TokenType
	Git                GitInfo
	Date               time.Time
	Artifacts          artifact.Artifacts
	ReleaseURL         string
	ReleaseNotes       string
	ReleaseNotesFile   string
	ReleaseNotesTmpl   string
	ReleaseHeaderFile  string
	ReleaseHeaderTmpl  string
	ReleaseFooterFile  string
	ReleaseFooterTmpl  string
	Version            string
	ModulePath         string
	Snapshot           bool
	SkipPostBuildHooks bool
	SkipPublish        bool
	SkipAnnounce       bool
	SkipSign           bool
	SkipValidate       bool
	SkipSBOMCataloging bool
	RmDist             bool
	PreRelease         bool
	Deprecated         bool
	Parallelism        int
	Semver             Semver
	Runtime            Runtime
}

type Runtime struct {
	Goos   string
	Goarch string
}

// Semver represents a semantic version.
type Semver struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	RawVersion string
	Prerelease string
}

// New context.
func New(config config.Project) *Context {
	return Wrap(ctx.Background(), config)
}

// NewWithTimeout new context with the given timeout.
func NewWithTimeout(config config.Project, timeout time.Duration) (*Context, ctx.CancelFunc) {
	ctx, cancel := ctx.WithTimeout(ctx.Background(), timeout)
	return Wrap(ctx, config), cancel
}

// Wrap wraps an existing context.
func Wrap(ctx ctx.Context, config config.Project) *Context {
	return &Context{
		Context:     ctx,
		Config:      config,
		Env:         ToEnv(append(os.Environ(), config.Env...)),
		Parallelism: 4,
		Artifacts:   artifact.New(),
		Date:        time.Now(),
		Runtime: Runtime{
			Goos:   runtime.GOOS,
			Goarch: runtime.GOARCH,
		},
	}
}

// ToEnv converts a list of strings to an Env (aka a map[string]string).
func ToEnv(env []string) Env {
	r := Env{}
	for _, e := range env {
		p := strings.SplitN(e, "=", 2)
		if len(p) != 2 || p[0] == "" {
			continue
		}
		r[p[0]] = p[1]
	}
	return r
}
