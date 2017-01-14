package git

import "github.com/goreleaser/releaser/context"

// Pipe for brew deployment
type Pipe struct{}

// Name of the pipe
func (Pipe) Name() string {
	return "Git"
}

// Run the pipe
func (Pipe) Run(context *context.Context) (err error) {
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

	context.Git.CurrentTag = tag
	context.Git.PreviousTag = previous
	context.Git.Diff = log
	return
}
