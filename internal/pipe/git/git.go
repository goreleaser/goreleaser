package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
)

// Pipe for brew deployment
type Pipe struct{}

func (Pipe) String() string {
	return "getting and validating git state"
}

// Run the pipe
func (Pipe) Run(ctx *context.Context) error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNoGit
	}
	info, err := getInfo(ctx)
	if err != nil {
		return err
	}
	ctx.Git = info
	log.Infof("releasing %s, commit %s", info.CurrentTag, info.Commit)
	if err := setVersion(ctx); err != nil {
		return err
	}
	return validate(ctx)
}

var fakeInfo = context.GitInfo{
	CurrentTag: "v0.0.0",
	Commit:     "none",
}

func getInfo(ctx *context.Context) (context.GitInfo, error) {
	if !git.IsRepo() && ctx.Snapshot {
		log.Warn("accepting to run without a git repo because this is a snapshot")
		return fakeInfo, nil
	}
	if !git.IsRepo() {
		return context.GitInfo{}, ErrNotRepository
	}
	info, err := getGitInfo(ctx)
	if err != nil && ctx.Snapshot {
		log.WithError(err).Warn("ignoring errors because this is a snapshot")
		if info.Commit == "" {
			info = fakeInfo
		}
		return info, nil
	}
	return info, err
}

func getGitInfo(ctx *context.Context) (context.GitInfo, error) {
	commit, err := getCommit(ctx)
	if err != nil {
		return context.GitInfo{}, errors.Wrap(err, "couldn't get current commit")
	}
	tag, err := getTag()
	if err != nil {
		return context.GitInfo{
			Commit:     commit,
			CurrentTag: "v0.0.0",
		}, ErrNoTag
	}
	return context.GitInfo{
		CurrentTag: tag,
		Commit:     commit,
	}, nil
}

func setVersion(ctx *context.Context) error {
	if ctx.Snapshot {
		snapshotName, err := tmpl.New(ctx).Apply(ctx.Config.Snapshot.NameTemplate)
		if err != nil {
			return errors.Wrap(err, "failed to generate snapshot name")
		}
		ctx.Version = snapshotName
		return nil
	}
	// removes usual `v` prefix
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return nil
}

func validate(ctx *context.Context) error {
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if ctx.SkipValidate {
		return pipe.ErrSkipValidateEnabled
	}
	out, err := git.Run("status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{status: out}
	}
	if !regexp.MustCompile("^[0-9.]+").MatchString(ctx.Version) {
		return ErrInvalidVersionFormat{version: ctx.Version}
	}
	_, err = git.Clean(git.Run("describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag))
	if err != nil {
		return ErrWrongRef{
			commit: ctx.Git.Commit,
			tag:    ctx.Git.CurrentTag,
		}
	}
	return nil
}

func getCommit(ctx *context.Context) (string, error) {
	format := "%H"
	if ctx.Config.Git.ShortHash {
		format = "%h"
	}
	return git.Clean(git.Run("show", fmt.Sprintf("--format='%s'", format), "HEAD"))
}

func getTag() (string, error) {
	return git.Clean(git.Run("describe", "--tags", "--abbrev=0"))
}
