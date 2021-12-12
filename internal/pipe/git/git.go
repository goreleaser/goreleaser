package git

import (
	"fmt"
	"net/url"
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
	log.WithField("commit", info.Commit).WithField("latest tag", info.CurrentTag).Info("building...")
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return validate(ctx)
}

// nolint: gochecknoglobals
var fakeInfo = context.GitInfo{
	Branch:      "none",
	CurrentTag:  "v0.0.0",
	Commit:      "none",
	ShortCommit: "none",
	FullCommit:  "none",
	Summary:     "none",
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
	branch, err := getBranch()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current branch: %w", err)
	}
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
	summary, err := getSummary()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get summary: %w", err)
	}
	gitURL, err := getURL()
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get remote URL: %w", err)
	}

	if strings.HasPrefix(gitURL, "https://") {
		u, err := url.Parse(gitURL)
		if err != nil {
			return context.GitInfo{}, fmt.Errorf("couldn't parse remote URL: %w", err)
		}
		u.User = nil
		gitURL = u.String()
	}

	tag, err := getTag()
	if err != nil {
		return context.GitInfo{
			Branch:      branch,
			Commit:      full,
			FullCommit:  full,
			ShortCommit: short,
			CommitDate:  date,
			URL:         gitURL,
			CurrentTag:  "v0.0.0",
			Summary:     summary,
		}, ErrNoTag
	}

	subject, err := getTagSubject(tag)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag subject: %w", err)
	}

	contents, err := getTagContents(tag)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag contents: %w", err)
	}

	previous, err := getPreviousTag(tag)
	if err != nil {
		// shouldn't error, will only affect templates
		log.Warnf("couldn't find any tags before %q", tag)
	}

	return context.GitInfo{
		Branch:      branch,
		CurrentTag:  tag,
		PreviousTag: previous,
		Commit:      full,
		FullCommit:  full,
		ShortCommit: short,
		CommitDate:  date,
		URL:         gitURL,
		Summary:     summary,
		TagSubject:  subject,
		TagContents: contents,
	}, nil
}

func validate(ctx *context.Context) error {
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if ctx.SkipValidate {
		return pipe.ErrSkipValidateEnabled
	}
	if _, err := os.Stat(".git/shallow"); err == nil {
		log.Warn("running against a shallow clone - check your CI documentation at https://goreleaser.com/ci")
	}
	if err := CheckDirty(); err != nil {
		return err
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

// CheckDirty returns an error if the current git repository is dirty.
func CheckDirty() error {
	out, err := git.Run("status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{status: out}
	}
	return nil
}

func getBranch() (string, error) {
	return git.Clean(git.Run("rev-parse", "--abbrev-ref", "HEAD", "--quiet"))
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

func getSummary() (string, error) {
	return git.Clean(git.Run("describe", "--always", "--dirty", "--tags"))
}

func getTagSubject(tag string) (string, error) {
	return git.Clean(git.Run("tag", "-l", "--format='%(contents:subject)'", tag))
}

func getTagContents(tag string) (string, error) {
	out, err := git.Run("tag", "-l", "--format='%(contents)'", tag)
	return strings.TrimSuffix(strings.ReplaceAll(out, "'", ""), "\n\n"), err
}

func getTag() (string, error) {
	var tag string
	var err error
	for _, fn := range []func() (string, error){
		func() (string, error) {
			return os.Getenv("GORELEASER_CURRENT_TAG"), nil
		},
		func() (string, error) {
			return git.Clean(git.Run("tag", "--points-at", "HEAD", "--sort", "-version:refname"))
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

func getPreviousTag(current string) (string, error) {
	if tag := os.Getenv("GORELEASER_PREVIOUS_TAG"); tag != "" {
		return tag, nil
	}

	return git.Clean(git.Run("describe", "--tags", "--abbrev=0", fmt.Sprintf("tags/%s^", current)))
}

func getURL() (string, error) {
	return git.Clean(git.Run("ls-remote", "--get-url"))
}
