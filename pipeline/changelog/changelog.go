// Package changelog provides the release changelog to goreleaser.
package changelog

import (
	"fmt"

	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/pipeline"
)

// Pipe for checksums
type Pipe struct{}

// Description of the pipe
func (Pipe) Description() string {
	return "Generating changelog"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if ctx.ReleaseNotes != "" {
		return pipeline.Skip("release notes already provided via --release-notes")
	}
	if ctx.Snapshot {
		return pipeline.Skip("not available for snapshots")
	}
	log, err := getChangelog(ctx.Git.CurrentTag)
	if err != nil {
		return err
	}
	ctx.ReleaseNotes = fmt.Sprintf("## Changelog\n\n%v", log)
	return nil
}

func getChangelog(tag string) (string, error) {
	prev, err := previous(tag)
	if err != nil {
		return "", err
	}
	if !prev.Tag {
		return gitLog(prev.SHA, tag)
	}
	return gitLog(fmt.Sprintf("%v..%v", prev.SHA, tag))
}

func gitLog(refs ...string) (string, error) {
	var args = []string{"log", "--pretty=oneline", "--abbrev-commit"}
	args = append(args, refs...)
	return git.Run(args...)
}

func previous(tag string) (result ref, err error) {
	result.Tag = true
	result.SHA, err = git.Clean(git.Run("describe", "--tags", "--abbrev=0", tag+"^"))
	if err != nil {
		result.Tag = false
		result.SHA, err = git.Clean(git.Run("rev-list", "--max-parents=0", "HEAD"))
	}
	return
}

type ref struct {
	Tag bool
	SHA string
}
