package git

import (
	"regexp"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

// ErrInvalidVersionFormat is return when the version isnt in a valid format
type ErrInvalidVersionFormat struct {
	version string
}

func (e ErrInvalidVersionFormat) Error() string {
	return e.version + " is not in a valid version format"
}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting Git info"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	tag, err := currentTag()
	if err != nil {
		return
	}
	previous, err := previousTag(tag)
	if err != nil {
		return
	}
	log, err := log(previous, tag)
	if err != nil {
		return
	}

	ctx.Git = context.GitInfo{
		CurrentTag:  tag,
		PreviousTag: previous,
		Diff:        log,
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(tag, "v")
	if matches, err := regexp.MatchString("[0-9.]+", ctx.Version); !matches || err != nil {
		return ErrInvalidVersionFormat{ctx.Version}
	}
	return
}
