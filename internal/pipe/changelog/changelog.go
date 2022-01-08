// Package changelog provides the release changelog to goreleaser.
package changelog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/git"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrInvalidSortDirection happens when the sort order is invalid.
var ErrInvalidSortDirection = errors.New("invalid sort direction")

const li = "* "

type useChangelog string

func (u useChangelog) formatable() bool {
	return u != "github-native"
}

const (
	useGit          = "git"
	useGitHub       = "github"
	useGitLab       = "gitlab"
	useGitHubNative = "github-native"
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string                 { return "generating changelog" }
func (Pipe) Skip(ctx *context.Context) bool { return ctx.Config.Changelog.Skip || ctx.Snapshot }

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	notes, err := loadContent(ctx, ctx.ReleaseNotesFile, ctx.ReleaseNotesTmpl)
	if err != nil {
		return err
	}
	ctx.ReleaseNotes = notes

	if ctx.ReleaseNotes != "" {
		return nil
	}

	footer, err := loadContent(ctx, ctx.ReleaseFooterFile, ctx.ReleaseFooterTmpl)
	if err != nil {
		return err
	}

	header, err := loadContent(ctx, ctx.ReleaseHeaderFile, ctx.ReleaseHeaderTmpl)
	if err != nil {
		return err
	}

	if err := checkSortDirection(ctx.Config.Changelog.Sort); err != nil {
		return err
	}

	entries, err := buildChangelog(ctx)
	if err != nil {
		return err
	}

	changes, err := formatChangelog(ctx, entries)
	if err != nil {
		return err
	}
	changelogElements := []string{changes}

	if header != "" {
		changelogElements = append([]string{header}, changelogElements...)
	}
	if footer != "" {
		changelogElements = append(changelogElements, footer)
	}

	ctx.ReleaseNotes = strings.Join(changelogElements, "\n\n")
	if !strings.HasSuffix(ctx.ReleaseNotes, "\n") {
		ctx.ReleaseNotes += "\n"
	}

	path := filepath.Join(ctx.Config.Dist, "CHANGELOG.md")
	log.WithField("changelog", path).Info("writing")
	return os.WriteFile(path, []byte(ctx.ReleaseNotes), 0o644) //nolint: gosec
}

func formatChangelog(ctx *context.Context, entries []string) (string, error) {
	newLine := "\n"
	if ctx.TokenType == context.TokenTypeGitLab || ctx.TokenType == context.TokenTypeGitea {
		// We need two or more whitespace to let markdown interpret
		// it as newline. See https://docs.gitlab.com/ee/user/markdown.html#newlines for details
		log.Debug("is gitlab or gitea changelog")
		newLine = "   \n"
	}

	if !useChangelog(ctx.Config.Changelog.Use).formatable() {
		return strings.Join(entries, newLine), nil
	}

	result := []string{"## Changelog"}
	if len(ctx.Config.Changelog.Groups) == 0 {
		log.Debug("not grouping entries")
		return strings.Join(append(result, filterAndPrefixItems(entries)...), newLine), nil
	}

	log.Debug("grouping entries")
	groups := ctx.Config.Changelog.Groups

	sort.Slice(groups, func(i, j int) bool { return groups[i].Order < groups[j].Order })
	for _, group := range groups {
		items := make([]string, 0)
		if group.Regexp == "" {
			// If no regexp is provided, we purge all strikethrough entries and add remaining entries to the list
			items = filterAndPrefixItems(entries)
			// clear array
			entries = nil
		} else {
			regex, err := regexp.Compile(group.Regexp)
			if err != nil {
				return "", fmt.Errorf("failed to group into %q: %w", group.Title, err)
			}
			for i, entry := range entries {
				match := regex.MatchString(entry)
				if match {
					items = append(items, li+entry)
					// Striking out the matched entry
					entries[i] = ""
				}
			}
		}
		if len(items) > 0 {
			result = append(result, fmt.Sprintf("### %s", group.Title))
			result = append(result, items...)
		}
	}

	return strings.Join(result, newLine), nil
}

func filterAndPrefixItems(ss []string) []string {
	var r []string
	for _, s := range ss {
		if s != "" {
			r = append(r, li+s)
		}
	}
	return r
}

func loadFromFile(file string) (string, error) {
	bts, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	log.WithField("file", file).Debugf("read %d bytes", len(bts))
	return string(bts), nil
}

func checkSortDirection(mode string) error {
	switch mode {
	case "":
		fallthrough
	case "asc":
		fallthrough
	case "desc":
		return nil
	}
	return ErrInvalidSortDirection
}

func buildChangelog(ctx *context.Context) ([]string, error) {
	log, err := getChangelog(ctx, ctx.Git.CurrentTag)
	if err != nil {
		return nil, err
	}
	entries := strings.Split(log, "\n")
	if lastLine := entries[len(entries)-1]; strings.TrimSpace(lastLine) == "" {
		entries = entries[0 : len(entries)-1]
	}
	if !useChangelog(ctx.Config.Changelog.Use).formatable() {
		return entries, nil
	}
	entries, err = filterEntries(ctx, entries)
	if err != nil {
		return entries, err
	}
	return sortEntries(ctx, entries), nil
}

func filterEntries(ctx *context.Context, entries []string) ([]string, error) {
	for _, filter := range ctx.Config.Changelog.Filters.Exclude {
		r, err := regexp.Compile(filter)
		if err != nil {
			return entries, err
		}
		entries = remove(r, entries)
	}
	return entries, nil
}

func sortEntries(ctx *context.Context, entries []string) []string {
	direction := ctx.Config.Changelog.Sort
	if direction == "" {
		return entries
	}
	result := make([]string, len(entries))
	copy(result, entries)
	sort.Slice(result, func(i, j int) bool {
		imsg := extractCommitInfo(result[i])
		jmsg := extractCommitInfo(result[j])
		if direction == "asc" {
			return strings.Compare(imsg, jmsg) < 0
		}
		return strings.Compare(imsg, jmsg) > 0
	})
	return result
}

func remove(filter *regexp.Regexp, entries []string) (result []string) {
	for _, entry := range entries {
		if !filter.MatchString(extractCommitInfo(entry)) {
			result = append(result, entry)
		}
	}
	return result
}

func extractCommitInfo(line string) string {
	return strings.Join(strings.Split(line, " ")[1:], " ")
}

func getChangelog(ctx *context.Context, tag string) (string, error) {
	prev := ctx.Git.PreviousTag
	if prev == "" {
		// get first commit
		result, err := git.Clean(git.Run("rev-list", "--max-parents=0", "HEAD"))
		if err != nil {
			return "", err
		}
		prev = result
	}
	return doGetChangelog(ctx, prev, tag)
}

func doGetChangelog(ctx *context.Context, prev, tag string) (string, error) {
	l, err := getChangeloger(ctx)
	if err != nil {
		return "", err
	}
	return l.Log(ctx, prev, tag)
}

func getChangeloger(ctx *context.Context) (changeloger, error) {
	switch ctx.Config.Changelog.Use {
	case useGit:
		fallthrough
	case "":
		return gitChangeloger{}, nil
	case useGitHub:
		fallthrough
	case useGitLab:
		return newSCMChangeloger(ctx)
	case useGitHubNative:
		return newGithubChangeloger(ctx)
	default:
		return nil, fmt.Errorf("invalid changelog.use: %q", ctx.Config.Changelog.Use)
	}
}

func newGithubChangeloger(ctx *context.Context) (changeloger, error) {
	cli, err := client.NewGitHub(ctx, ctx.Token)
	if err != nil {
		return nil, err
	}
	repo, err := git.ExtractRepoFromConfig()
	if err != nil {
		return nil, err
	}
	return &githubNativeChangeloger{
		client: cli,
		repo: client.Repo{
			Owner: repo.Owner,
			Name:  repo.Name,
		},
	}, nil
}

func newSCMChangeloger(ctx *context.Context) (changeloger, error) {
	cli, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := git.ExtractRepoFromConfig()
	if err != nil {
		return nil, err
	}
	return &scmChangeloger{
		client: cli,
		repo: client.Repo{
			Owner: repo.Owner,
			Name:  repo.Name,
		},
	}, nil
}

func loadContent(ctx *context.Context, fileName, tmplName string) (string, error) {
	if tmplName != "" {
		log.Debugf("loading template %s", tmplName)
		content, err := loadFromFile(tmplName)
		if err != nil {
			return "", err
		}
		return tmpl.New(ctx).Apply(content)
	}

	if fileName != "" {
		log.Debugf("loading file %s", fileName)
		return loadFromFile(fileName)
	}

	return "", nil
}

type changeloger interface {
	Log(ctx *context.Context, prev, current string) (string, error)
}

type gitChangeloger struct{}

var validSHA1 = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)

func (g gitChangeloger) Log(_ *context.Context, prev, current string) (string, error) {
	args := []string{"log", "--pretty=oneline", "--abbrev-commit", "--no-decorate", "--no-color"}
	if validSHA1.MatchString(prev) {
		args = append(args, prev, current)
	} else {
		args = append(args, fmt.Sprintf("tags/%s..tags/%s", prev, current))
	}
	return git.Run(args...)
}

type scmChangeloger struct {
	client client.Client
	repo   client.Repo
}

func (c *scmChangeloger) Log(ctx *context.Context, prev, current string) (string, error) {
	return c.client.Changelog(ctx, c.repo, prev, current)
}

type githubNativeChangeloger struct {
	client client.GitHubClient
	repo   client.Repo
}

func (c *githubNativeChangeloger) Log(ctx *context.Context, prev, current string) (string, error) {
	return c.client.GenerateReleaseNotes(ctx, c.repo, prev, current)
}
