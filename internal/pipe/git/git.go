package git

import (
	"os/exec"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
)

// Pipe that sets up git state
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
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return validate(ctx)
}

// nolint: gochecknoglobals
var fakeInfo = context.GitInfo{
	CurrentTag:  "v0.0.0",
	Commit:      "none",
	ShortCommit: "none",
	FullCommit:  "none",
}

func getInfo(ctx *context.Context) (context.GitInfo, error) {
	if !git.IsRepo() && ctx.Snapshot {
		log.Warn("accepting to run without a git repo because this is a snapshot")
		return fakeInfo, nil
	}
	if !git.IsRepo() {
		return context.GitInfo{}, ErrNotRepository
	}
	info, err := getGitInfo()
	if err != nil && ctx.Snapshot {
		log.WithError(err).Warn("ignoring errors because this is a snapshot")
		if info.Commit == "" {
			info = fakeInfo
		}
		return info, nil
	}
	return info, err
}

func getGitInfo() (context.GitInfo, error) {
	short, err := getShortCommit()
	if err != nil {
		return context.GitInfo{}, errors.Wrap(err, "couldn't get current commit")
	}
	full, err := getFullCommit()
	if err != nil {
		return context.GitInfo{}, errors.Wrap(err, "couldn't get current commit")
	}
	url, err := getURL()
	if err != nil {
		return context.GitInfo{}, errors.Wrap(err, "couldn't get remote URL")
	}
	tag, err := getTag()
	if err != nil || tag == "" {
		return context.GitInfo{
			Commit:      full,
			FullCommit:  full,
			ShortCommit: short,
			URL:         url,
			CurrentTag:  "v0.0.0",
		}, ErrNoTag
	}
	return context.GitInfo{
		CurrentTag:  tag,
		Commit:      full,
		FullCommit:  full,
		ShortCommit: short,
		URL:         url,
	}, nil
}

func validate(ctx *context.Context) error {
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if ctx.SkipValidate {
		return pipe.ErrSkipValidateEnabled
	}
	if hasDiff() {
		return diffErr()
	}
	_, err := git.Clean(git.Run("describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag))
	if err != nil {
		return ErrWrongRef{
			commit: ctx.Git.Commit,
			tag:    ctx.Git.CurrentTag,
		}
	}
	return nil
}

func diffErr() error {
	out, err := git.Run("diff")
	if err != nil {
		// in theory this will never happen...
		return ErrDirty{status: err.Error()}
	}
	return ErrDirty{status: out}
}

func hasDiff() bool {
	out, err := git.Run("status", "--porcelain")
	return strings.TrimSpace(out) != "" || err != nil
}

func getShortCommit() (string, error) {
	return git.Clean(git.Run("show", "--format='%h'", "HEAD", "-q"))
}

func getFullCommit() (string, error) {
	return git.Clean(git.Run("show", "--format='%H'", "HEAD", "-q"))
}

func getTag() (string, error) {
	// Even when version sort is used in git-tag[1], tagnames with the same base version
	// but different suffixes are still sorted lexicographically, resulting e.g. in prerelease tags
	// appearing after the main release (e.g. "1.0-rc1" after "1.0").
	// This variable can be specified to determine the sorting order of tags with different suffixes.
	// https://git-scm.com/docs/git-config/2.19.2#git-config-versionsortsuffix
	return git.Clean(git.Run("-c", "versionsort.suffix=-", "tag", "-l", "--sort=-v:refname"))
}

func getURL() (string, error) {
	return git.Clean(git.Run("ls-remote", "--get-url"))
}
