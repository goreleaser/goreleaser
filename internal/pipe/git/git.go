package git

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/apex/log"

	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Pipe that sets up git state.
type Pipe struct{}

func (Pipe) String() string {
	return "getting and validating git state"
}

// Run the pipe.
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
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	full, err := getFullCommit()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	date, err := getCommitDate()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get commit date: %w", err)
	}
	url, err := getURL()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get remote URL: %w", err)
	}
	tag, err := getTag()
	if err != nil {
		return context.GitInfo{
			Commit:      full,
			FullCommit:  full,
			ShortCommit: short,
			CommitDate:  date,
			URL:         url,
			CurrentTag:  "v0.0.0",
		}, ErrNoTag
	}
	return context.GitInfo{
		CurrentTag:  tag,
		Commit:      full,
		FullCommit:  full,
		ShortCommit: short,
		CommitDate:  date,
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
	out, err := git.Run("status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{status: out}
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

func getCommitDate() (time.Time, error) {
	ct, err := git.Clean(git.Run("show", "--format='%ct'", "HEAD", "--quiet"))
	if err != nil {
		return time.Time{}, err
	}
	if ct == "" {
		return time.Time{}, nil
	}
	i, err := strconv.ParseInt(ct, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	t := time.Unix(i, 0).UTC()
	return t, nil
}

func getShortCommit() (string, error) {
	return git.Clean(git.Run("show", "--format='%h'", "HEAD", "--quiet"))
}

func getFullCommit() (string, error) {
	return git.Clean(git.Run("show", "--format='%H'", "HEAD", "--quiet"))
}

func getTag() (string, error) {
	var tag string
	var err error
	for _, fn := range []func() (string, error){
		func() (string, error) {
			return os.Getenv("GORELEASER_CURRENT_TAG"), nil
		},
		func() (string, error) {
			return git.Clean(git.Run("tag", "--points-at", "HEAD", "--sort", "-version:creatordate"))
		},
		func() (string, error) {
			return git.Clean(git.Run("describe", "--tags", "--abbrev=0"))
		},
	} {
		tag, err = fn()
		if tag != "" || err != nil {
			return tag, err
		}
	}

	return tag, err
}

func getURL() (string, error) {
	return git.Clean(git.Run("ls-remote", "--get-url"))
}
