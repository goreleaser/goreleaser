// Package source provides pipes to take care of validating the current
// git repo state.
// For the releasing process we need the files of the tag we are releasing.
package source

import (
	"os/exec"

	"github.com/goreleaser/goreleaser/context"
)

// ErrDirty happens when the repo has uncommitted/unstashed changes
type ErrDirty struct {
	status string
}

func (e ErrDirty) Error() string {
	return "git is currently in a dirty state: " + e.status
}

// ErrWrongRef happens when the HEAD reference is different from the tag being built
type ErrWrongRef struct {
	status string
}

func (e ErrWrongRef) Error() string {
	return "current tag ref is different from HEAD ref: " + e.status
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
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	if err := cmd.Run(); err != nil {
		status, err := status()
		if err != nil {
			return err
		}
		return ErrDirty{status}
	}

	cmd = exec.Command("git", "describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag)
	if err := cmd.Run(); err != nil {
		status, err := status()
		if err != nil {
			return err
		}
		return ErrWrongRef{status}
	}
	return nil
}

func status() (string, error) {
	bts, err := exec.Command("git", "status", "-sb").CombinedOutput()
	if err != nil {
		return "", err
	}
	return "\n\n" + string(bts), nil
}
