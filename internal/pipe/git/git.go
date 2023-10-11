package git

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/x/exp/ordered"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
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
		WithField("current_tag", info.CurrentTag).
		WithField("previous_tag", ordered.First(info.PreviousTag, "<unknown>")).
		WithField("dirty", info.Dirty).
		Info("git state")
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
	if !git.IsRepo(ctx) && ctx.Snapshot {
		log.Warn("accepting to run without a git repository because this is a snapshot")
		return fakeInfo, nil
	}
	if !git.IsRepo(ctx) {
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
	short, err := getShortCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	full, err := getFullCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get current commit: %w", err)
	}
	first, err := getFirstCommit(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get first commit: %w", err)
	}
	date, err := getCommitDate(ctx)
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get commit date: %w", err)
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
		}, ErrNoTag
	}

	subject, err := getTagWithFormat(ctx, tag, "contents:subject")
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag subject: %w", err)
	}

	contents, err := getTagWithFormat(ctx, tag, "contents")
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag contents: %w", err)
	}

	body, err := getTagWithFormat(ctx, tag, "contents:body")
	if err != nil {
		return context.GitInfo{}, fmt.Errorf("couldn't get tag content body: %w", err)
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
	return git.Clean(git.Run(ctx, "rev-parse", "--abbrev-ref", "HEAD", "--quiet"))
}

func getCommitDate(ctx *context.Context) (time.Time, error) {
	ct, err := git.Clean(git.Run(ctx, "show", "--format='%ct'", "HEAD", "--quiet"))
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

func getShortCommit(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "show", "--format=%h", "HEAD", "--quiet"))
}

func getFullCommit(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "show", "--format=%H", "HEAD", "--quiet"))
}

func getFirstCommit(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "rev-list", "--max-parents=0", "HEAD"))
}

func getSummary(ctx *context.Context) (string, error) {
	return git.Clean(git.Run(ctx, "describe", "--always", "--dirty", "--tags"))
}

func getTagWithFormat(ctx *context.Context, tag, format string) (string, error) {
	out, err := git.Run(ctx, "tag", "-l", "--format='%("+format+")'", tag)
	return strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(out, "'", ""), "\n\n")), err
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
		for _, exl := range exclude {
			if exl != tag {
				return tag
			}
		}
	}
	return ""
}
