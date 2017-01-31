// Package source provides pipes to take care of validating the current
// git repo state.
// For the releasing process we need the files of the tag we are releasing.
package source

import (
	"os/exec"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

// ErrDirty happens when the repo has uncommitted/unstashed changes
type ErrDirty struct {
	status string
}

func (e ErrDirty) Error() string {
	return "git is currently in a dirty state:\n" + e.status
}

// ErrWrongRef happens when the HEAD reference is different from the tag being built
type ErrWrongRef struct {
	status string
}

func (e ErrWrongRef) Error() string {
	return e.status
}

// Pipe to make sure we are in the latest Git tag as source.
type Pipe struct{}

// Description of the pipe
func (p Pipe) Description() string {
	return "Validating current git state"
}

// Run errors we the repo is dirty or if the current ref is different from the
// tag ref
func (p Pipe) Run(ctx *context.Context) error {
	bts, err := exec.Command("git", "diff").CombinedOutput()
	if err != nil || strings.TrimSpace(string(bts)) != "" {
		return ErrDirty{string(bts)}
	}

	cmd := exec.Command("git", "describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag)
	if bts, err := cmd.CombinedOutput(); err != nil {
		return ErrWrongRef{string(bts)}
	}
	return nil
}
