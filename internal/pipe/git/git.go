package git

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Pipe that sets up git state.
type Pipe struct{}

func (Pipe) String() string {
	return "getting and validating git state"
}

// this pipe does not implement Defaulter because it runs before the defaults
// pipe, and we need to set some defaults of our own first.
func setDefaults(ctx *context.Context) {
	if ctx.Config.Git.TagSort == "" {
		ctx.Config.Git.TagSort = "-version:refname"
	}
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	if _, err := exec.LookPath("git"); err != nil {
		return ErrNoGit
	}
	setDefaults(ctx)
	info, err := getInfo(ctx)
	if err != nil {
		return err
	}
	ctx.Git = info
	log.WithField("commit", info.Commit).
		WithField("branch", info.Branch).
		WithField("previous_tag", cmp.Or(info.PreviousTag, "<unknown>")).
		WithField("current_tag", info.CurrentTag).
		WithField("dirty", info.Dirty).
		Debug("git state")
	log.
		WithField("previous", cmp.Or(info.PreviousTag, "<unknown>")).
		WithField("current", info.CurrentTag).
		Info("using tags")
	ctx.Version = strings.TrimPrefix(ctx.Git.CurrentTag, "v")
	return validate(ctx)
}

//nolint:gochecknoglobals
var fakeInfo = context.GitInfo{
	Branch:      "none",
	CurrentTag:  "v0.0.0",
	Commit:      "none",
	ShortCommit: "none",
	FullCommit:  "none",
	Summary:     "none",
}

func getInfo(ctx *context.Context) (context.GitInfo, error) {
	if !git.IsRepo(ctx) {
		if ctx.Snapshot {
			log.Warn("accepting to run without a git repository because this is a snapshot")
			return fakeInfo, nil
		}
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
	branch, err := getBranch(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current branch: %w", err)
	}
	short, full, date, err := getCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	first, err := getFirstCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get first commit: %w", err)
	}
	summary, err := getSummary(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get summary: %w", err)
	}
	gitURL, err := getURL(ctx)
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

	var excluding []string
	tpl := tmpl.New(ctx)
	for _, exclude := range ctx.Config.Git.IgnoreTags {
		tag, err := tpl.Apply(exclude)
		if err != nil {
			return context.GitInfo{}, err
		}
		excluding = append(excluding, tag)
	}

	tag, err := getTag(ctx, excluding)
	if err != nil {
		return context.GitInfo{
			Branch:      branch,
			Commit:      full,
			FullCommit:  full,
			ShortCommit: short,
			FirstCommit: first,
			CommitDate:  date,
			URL:         gitURL,
			CurrentTag:  "v0.0.0",
			Summary:     summary,
			Dirty:       CheckDirty(ctx) != nil,
		}, ErrNoTag
	}

	subject, contents, body, err := getTagContents(ctx, tag)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag contents: %w", err)
	}

	previous, err := getPreviousTag(ctx, tag, excluding)
	if err != nil {
		// shouldn't error, will only affect templates and changelog
		log.Warnf("couldn't find any tags before %q", tag)
	}

	return context.GitInfo{
		Branch:      branch,
		CurrentTag:  tag,
		PreviousTag: previous,
		Commit:      full,
		FullCommit:  full,
		ShortCommit: short,
		FirstCommit: first,
		CommitDate:  date,
		URL:         gitURL,
		Summary:     summary,
		TagSubject:  subject,
		TagContents: contents,
		TagBody:     body,
		Dirty:       CheckDirty(ctx) != nil,
	}, nil
}

func validate(ctx *context.Context) error {
	if ctx.Snapshot {
		return pipe.ErrSnapshotEnabled
	}
	if skips.Any(ctx, skips.Validate) {
		return pipe.ErrSkipValidateEnabled
	}
	if _, err := os.Stat(".git/shallow"); err == nil {
		log.Warn("running against a shallow clone - check your CI documentation at https://goreleaser.com/ci")
	}
	if err := CheckDirty(ctx); err != nil {
		return err
	}
	_, err := git.Clean(git.Run(ctx, "describe", "--exact-match", "--tags", "--match", ctx.Git.CurrentTag))
	if err != nil {
		return ErrWrongRef{
			commit: ctx.Git.Commit,
			tag:    ctx.Git.CurrentTag,
		}
	}
	return nil
}

// CheckDirty returns an error if the current git repository is dirty.
func CheckDirty(ctx *context.Context) error {
	out, err := git.Run(ctx, "status", "--porcelain")
	if strings.TrimSpace(out) != "" || err != nil {
		return ErrDirty{status: out}
	}
	return nil
}

func getBranch(ctx *context.Context) (string, error) {
	out, err := git.Run(ctx, "rev-parse", "--abbrev-ref", "HEAD", "--quiet")
	if err != nil {
		return "", errors.New(strings.TrimSuffix(err.Error(), "\n"))
	}
	return strings.TrimRight(out, "\r\n"), nil
}

// getCommit returns the short hash, full hash and committer date of HEAD.
//
// One `git show` rather than three that differ only in --format: process
// creation is expensive on windows, and every run of this pipe lands here.
func getCommit(ctx *context.Context) (short, full string, date time.Time, err error) {
	out, err := git.Run(ctx, "show", "--format=%h%n%H%n%ct", "HEAD", "--quiet")
	if err != nil {
		return "", "", time.Time{}, err
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		return "", "", time.Time{}, fmt.Errorf("unexpected git show output: %q", out)
	}
	short, full, ct := lines[0], lines[1], lines[2]
	if ct == "" {
		return short, full, time.Time{}, nil
	}
	i, err := strconv.ParseInt(ct, 10, 64)
	if err != nil {
		return "", "", time.Time{}, err
	}
	return short, full, time.Unix(i, 0).UTC(), nil
}

func getFirstCommit(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "rev-list", "--max-parents=0", "HEAD"))
}

func getSummary(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "describe", "--always", "--dirty", "--tags"))
}

// getTagContents returns the subject, full contents and body of the given
// tag's message.
//
// One `git tag -l` rather than three that differ only in the format: the three
// fields are separated by NUL, which a tag message cannot contain. The format
// is not quoted on purpose: git would print the quotes verbatim, and stripping
// them afterwards would also strip any apostrophes the message contains.
func getTagContents(ctx *context.Context, tag string) (subject, contents, body string, err error) {
	out, err := git.Run(ctx, "tag", "-l", "--format=%(contents:subject)%00%(contents)%00%(contents:body)", tag)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.Split(out, "\x00")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected git tag output for %q: %q", tag, out)
	}
	return strings.TrimSpace(parts[0]),
		strings.TrimSpace(parts[1]),
		strings.TrimSpace(parts[2]),
		nil
}

func getTag(ctx *context.Context, excluding []string) (string, error) {
	for _, fn := range []func() ([]string, error){
		getFromEnv("GORELEASER_CURRENT_TAG"),
		func() ([]string, error) {
			return gitTagsPointingAt(ctx, "HEAD")
		},
		func() ([]string, error) {
			// this will get the last tag, even if it wasn't made against the
			// last commit...
			return git.CleanAllLines(gitDescribe(ctx, "HEAD", excluding))
		},
	} {
		tags, err := fn()
		if err != nil {
			return "", err
		}
		if tag := filterOut(tags, excluding); tag != "" {
			return tag, err
		}
	}

	return "", nil
}

func getPreviousTag(ctx *context.Context, current string, excluding []string) (string, error) {
	for _, fn := range []func() ([]string, error){
		getFromEnv("GORELEASER_PREVIOUS_TAG"),
		func() ([]string, error) {
			sha, err := previousTagSha(ctx, current, excluding)
			if err != nil {
				return nil, err
			}
			return gitTagsPointingAt(ctx, sha)
		},
	} {
		tags, err := fn()
		if err != nil {
			return "", err
		}
		if tag := filterOut(tags, excluding); tag != "" {
			return tag, nil
		}
	}

	return "", nil
}

func gitTagsPointingAt(ctx *context.Context, ref string) ([]string, error) {
	args := []string{}
	if ctx.Config.Git.PrereleaseSuffix != "" {
		args = append(
			args,
			"-c",
			"versionsort.suffix="+ctx.Config.Git.PrereleaseSuffix,
		)
	}
	args = append(
		args,
		"tag",
		"--points-at",
		ref,
		"--sort",
		ctx.Config.Git.TagSort,
	)
	return git.CleanAllLines(git.Run(ctx, args...))
}

func gitDescribe(ctx *context.Context, ref string, excluding []string) (string, error) {
	args := []string{
		"describe",
		"--tags",
		"--abbrev=0",
		ref,
	}
	for _, exclude := range excluding {
		args = append(args, "--exclude="+exclude)
	}
	return git.Clean(git.Run(ctx, args...))
}

func previousTagSha(ctx *context.Context, current string, excluding []string) (string, error) {
	tag, err := gitDescribe(ctx, fmt.Sprintf("tags/%s^", current), excluding)
	if err != nil {
		return "", err
	}
	return git.Clean(git.Run(ctx, "rev-list", "-n1", tag))
}

func getURL(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "ls-remote", "--get-url"))
}

func getFromEnv(s string) func() ([]string, error) {
	return func() ([]string, error) {
		if tag := os.Getenv(s); tag != "" {
			return []string{tag}, nil
		}
		return nil, nil
	}
}

func filterOut(tags []string, exclude []string) string {
	if len(exclude) == 0 && len(tags) > 0 {
		return tags[0]
	}
	for _, tag := range tags {
		if !slices.Contains(exclude, tag) {
			return tag
		}
	}
	return ""
}
