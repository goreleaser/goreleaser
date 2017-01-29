package git

import (
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

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
	return
}
