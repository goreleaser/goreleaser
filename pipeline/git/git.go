// Package git implements the Pipe interface extracting usefull data from
// git and putting it in the context.
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

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	tag, err := cleanGit("describe", "--tags", "--abbrev=0", "--always")
	if err != nil {
		return
	}
	prev, err := previous(tag)
	if err != nil {
		return
	}

	log, err := git("log", "--pretty=oneline", "--abbrev-commit", prev+".."+tag)
	if err != nil {
		return
	}

	ctx.Git = context.GitInfo{
		CurrentTag:  tag,
		PreviousTag: prev,
		Diff:        log,
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(tag, "v")
	if versionErr := isVersionValid(ctx.Version); versionErr != nil {
		return versionErr
	}
	commit, err := cleanGit("show", "--format='%H'", "HEAD")
	if err != nil {
		return
	}
	ctx.Git.Commit = commit
	out, err := git("diff")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{out}
	}
	_, err = cleanGit("describe", "--exact-match", "--tags", "--match", tag)
	if err != nil {
		return ErrWrongRef{err.Error()}
	}
	return nil
}

func previous(tag string) (previous string, err error) {
	previous, err = cleanGit("describe", "--tags", "--abbrev=0", "--always", tag+"^")
	if err != nil {
		previous, err = cleanGit("rev-list", "--max-parents=0", "HEAD")
	}
	return
}

func isVersionValid(version string) error {
	matches, err := regexp.MatchString("^[0-9.]+", version)
	if err != nil || !matches {
		return ErrInvalidVersionFormat{version}
	}
	return nil
}
