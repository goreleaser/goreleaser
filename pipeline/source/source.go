// Package source provides pipes to take care of validating the current
// git repo state.
// For the releasing process we need the files of the tag we are releasing.
package source

import (
	"errors"
	"os/exec"

	"github.com/goreleaser/goreleaser/context"
)

// ErrDirty happens when the repo has uncommitted/unstashed changes
var ErrDirty = errors.New("git is currently in a dirty state, commit or stash your changes to continue")

var ErrWrongRef = errors.New("current tag ref is different from HEAD ref")

// Pipe to make sure we are in the latest Git tag as source.
type Pipe struct{}

// Description of the pipe
func (p *Pipe) Description() string {
	return "Validating current git state"
}

// Run errors we the repo is dirty or if the current ref is different from the
// tag ref
func (p *Pipe) Run(ctx *context.Context) error {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	if err := cmd.Run(); err != nil {
		return ErrDirty
	}

	cmd = exec.Command("git", "describe", "--exact-match", "--match", ctx.Git.CurrentTag)
	if err := cmd.Run(); err != nil {
		return ErrWrongRef
	}
	return nil
}
