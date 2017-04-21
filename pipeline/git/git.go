// Package git implements the Pipe interface getting and validating the
// current git repository state
package git

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/goreleaser/goreleaser/context"
)

// ErrInvalidVersionFormat is return when the version isnt in a valid format
type ErrInvalidVersionFormat struct {
	version string
}

func (e ErrInvalidVersionFormat) Error() string {
	return fmt.Sprintf("%v is not in a valid version format", e.version)
}

// ErrDirty happens when the repo has uncommitted/unstashed changes
type ErrDirty struct {
	status string
}

func (e ErrDirty) Error() string {
	return fmt.Sprintf("git is currently in a dirty state:\n%v", e.status)
}

// ErrWrongRef happens when the HEAD reference is different from the tag being built
type ErrWrongRef struct {
	commit, tag string
}

func (e ErrWrongRef) Error() string {
	return fmt.Sprintf("git tag %v was not made against commit %v", e.tag, e.commit)
}

// Pipe for brew deployment
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) (err error) {
	tag, commit, err := getInfo()
	if err != nil {
		return
	}
	ctx.Git = context.GitInfo{
		CurrentTag: tag,
		Commit:     commit,
	}
	if ctx.ReleaseNotes == "" {
		log, err := getChangelog(tag)
		if err != nil {
			return err
		}
		ctx.ReleaseNotes = fmt.Sprintf("## Changelog\n\n%v", log)
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(tag, "v")
	if !ctx.Validate {
		log.Println("Skipped validations because --skip-validate is set")
		return nil
	}
	return validate(commit, tag, ctx.Version)
}

func validate(commit, tag, version string) error {
	matches, err := regexp.MatchString("^[0-9.]+", version)
	if err != nil || !matches {
		return ErrInvalidVersionFormat{version}
	}
	out, err := git("status", "-s")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{out}
	}
	_, err = cleanGit("describe", "--exact-match", "--tags", "--match", tag)
	if err != nil {
		return ErrWrongRef{commit, tag}
	}
	return nil
}

func getChangelog(tag string) (string, error) {
	prev, err := previous(tag)
	if err != nil {
		return "", err
	}
	return git("log", "--pretty=oneline", "--abbrev-commit", prev+".."+tag)
}

func getInfo() (tag, commit string, err error) {
	tag, err = cleanGit("describe", "--tags", "--abbrev=0", "--always")
	if err != nil {
		return
	}
	commit, err = cleanGit("show", "--format='%H'", "HEAD")
	return
}

func previous(tag string) (previous string, err error) {
	previous, err = cleanGit("describe", "--tags", "--abbrev=0", "--always", tag+"^")
	if err != nil {
		previous, err = cleanGit("rev-list", "--max-parents=0", "HEAD")
	}
	return
}
