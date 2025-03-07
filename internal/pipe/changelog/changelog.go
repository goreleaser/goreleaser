// Package changelog provides the release changelog to goreleaser.
package changelog

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/git"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// ErrInvalidSortDirection happens when the sort order is invalid.
var ErrInvalidSortDirection = errors.New("invalid sort direction")

const li = "* "

type useChangelog string

const (
	useGit          = "git"
	useGitHub       = "github"
	useGitea        = "gitea"
	useGitLab       = "gitlab"
	useGitHubNative = "github-native"
)

// Pipe for checksums.
type Pipe struct{}

func (Pipe) String() string { return "generating changelog" }

func (Pipe) Skip(ctx *context.Context) (bool, error) {
	if ctx.Snapshot {
		return true, nil
	}

	return tmpl.New(ctx).Bool(ctx.Config.Changelog.Disable)
}

func (Pipe) Default(ctx *context.Context) error {
	if ctx.Config.Changelog.Format == "" {
		switch ctx.Config.Changelog.Use {
		case "", "git":
			ctx.Config.Changelog.Format = "{{ .SHA }} {{ .Message }}"
		default:
			ctx.Config.Changelog.Format = "{{ .SHA }}: {{ .Message }} ({{ with .AuthorUsername }}@{{ . }}{{ else }}{{ .AuthorName }} <{{ .AuthorEmail }}>{{ end }})"
		}
	}
	return nil
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	notes, err := loadContent(ctx, ctx.ReleaseNotesFile, ctx.ReleaseNotesTmpl)
	if err != nil {
		return err
	}
	ctx.ReleaseNotes = notes

	if ctx.ReleaseNotesFile != "" || ctx.ReleaseNotesTmpl != "" {
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

	out, err := getChangelog(ctx)
	if err != nil {
		return err
	}
	changelogElements := []string{out}

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
	log.WithField("path", path).Debug("writing changelog")
	return os.WriteFile(path, []byte(ctx.ReleaseNotes), 0o644) //nolint:gosec
}

func getChangelog(ctx *context.Context) (string, error) {
	if ctx.Config.Changelog.Use == "github-native" {
		l, err := newGithubChangeloger(ctx)
		if err != nil {
			return "", err
		}
		return l.Log(ctx)
	}

	if err := checkSortDirection(ctx.Config.Changelog.Sort); err != nil {
		return "", err
	}

	entries, err := buildChangelog(ctx)
	if err != nil {
		return "", err
	}

	return formatChangelog(ctx, entries)
}

func applyFormatTo(ctx *context.Context, entries ...client.ChangelogItem) ([]string, error) {
	var lines []string
	for _, item := range entries {
		line, err := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
			"SHA":            abbrevEntry(item.SHA, ctx.Config.Changelog.Abbrev),
			"Message":        item.Message,
			"AuthorUsername": item.AuthorUsername,
			"AuthorName":     item.AuthorName,
			"AuthorEmail":    item.AuthorEmail,
		}).Apply(ctx.Config.Changelog.Format)
		if err != nil {
			return nil, err
		}
		lines = append(lines, prefixItem(line))
	}
	return lines, nil
}

type changelogGroup struct {
	title   string
	entries []string
	order   int
}

func title(s string, level int) string {
	if s == "" {
		return ""
	}
	return fmt.Sprintf("%s %s", strings.Repeat("#", level), s)
}

func newLineFor(ctx *context.Context) string {
	if ctx.TokenType == context.TokenTypeGitLab || ctx.TokenType == context.TokenTypeGitea {
		// We need two or more whitespace to let markdown interpret
		// it as newline. See https://docs.gitlab.com/ee/user/markdown.html#newlines for details
		log.Debug("is gitlab or gitea changelog")
		return "   \n"
	}

	return "\n"
}

func abbrevEntry(sha string, abbr int) string {
	switch abbr {
	case 0:
		return sha
	case -1:
		return ""
	default:
		if abbr > len(sha) {
			return sha
		}
		return sha[:abbr]
	}
}

func formatChangelog(ctx *context.Context, entries []client.ChangelogItem) (string, error) {
	result := []string{title("Changelog", 2)}
	if len(ctx.Config.Changelog.Groups) == 0 {
		log.Debug("not grouping entries")
		lines, err := applyFormatTo(ctx, entries...)
		return strings.Join(append(result, lines...), newLineFor(ctx)), err
	}

	log.Debug("grouping entries")
	var groups []changelogGroup
	for _, group := range ctx.Config.Changelog.Groups {
		item := changelogGroup{
			title: title(group.Title, 3),
			order: group.Order,
		}
		if group.Regexp == "" {
			// If no regexp is provided, we purge all strikethrough entries and add remaining entries to the list
			lines, err := applyFormatTo(ctx, entries...)
			if err != nil {
				return "", err
			}
			item.entries = lines
			// clear array
			entries = nil
		} else {
			re, err := regexp.Compile(group.Regexp)
			if err != nil {
				return "", fmt.Errorf("failed to group into %q: %w", group.Title, err)
			}

			log.Debugf("group: %#v", group)
			i := 0
			for _, entry := range entries {
				match := re.MatchString(entry.Message)
				log.Debugf("entry: %s match: %b\n", entry, match)
				if match {
					line, err := applyFormatTo(ctx, entry)
					if err != nil {
						return "", err
					}
					item.entries = append(item.entries, line...)
				} else {
					// Keep unmatched entry.
					entries[i] = entry
					i++
				}
			}
			entries = entries[:i]
		}
		groups = append(groups, item)

		if len(entries) == 0 {
			break // No more entries to process.
		}
	}

	slices.SortFunc(groups, groupSort)
	for _, group := range groups {
		if len(group.entries) > 0 {
			result = append(result, group.title)
			result = append(result, group.entries...)
		}
	}
	return strings.Join(result, newLineFor(ctx)), nil
}

func groupSort(i, j changelogGroup) int {
	return cmp.Compare(i.order, j.order)
}

func prefixItem(s string) string {
	if s == "" {
		return s
	}
	return li + s
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
	case "", "asc", "desc":
		return nil
	default:
		return ErrInvalidSortDirection
	}
}

func buildChangelog(ctx *context.Context) ([]client.ChangelogItem, error) {
	l, err := getFormatableChangeloger(ctx)
	if err != nil {
		return nil, err
	}
	entries, err := l.Log(ctx)
	if err != nil {
		return nil, err
	}
	entries, err = filterEntries(ctx, entries)
	if err != nil {
		return entries, err
	}
	return sortEntries(ctx, entries), nil
}

func filterEntries(ctx *context.Context, entries []client.ChangelogItem) ([]client.ChangelogItem, error) {
	filters := ctx.Config.Changelog.Filters
	if len(filters.Include) > 0 {
		var newEntries []client.ChangelogItem
		for _, filter := range filters.Include {
			r, err := regexp.Compile(filter)
			if err != nil {
				return entries, err
			}
			newEntries = append(newEntries, keep(r, entries)...)
		}
		return newEntries, nil
	}
	for _, filter := range filters.Exclude {
		r, err := regexp.Compile(filter)
		if err != nil {
			return entries, err
		}
		entries = remove(r, entries)
	}
	return entries, nil
}

func sortEntries(ctx *context.Context, entries []client.ChangelogItem) []client.ChangelogItem {
	direction := ctx.Config.Changelog.Sort
	if direction == "" {
		return entries
	}
	slices.SortFunc(entries, func(i, j client.ChangelogItem) int {
		compareRes := strings.Compare(i.Message, j.Message)
		if direction == "asc" {
			return compareRes
		}
		return -compareRes
	})
	return entries
}

func keep(filter *regexp.Regexp, entries []client.ChangelogItem) (result []client.ChangelogItem) {
	for _, entry := range entries {
		if filter.MatchString(entry.Message) {
			result = append(result, entry)
		}
	}
	return result
}

func remove(filter *regexp.Regexp, entries []client.ChangelogItem) (result []client.ChangelogItem) {
	for _, entry := range entries {
		if !filter.MatchString(entry.Message) {
			result = append(result, entry)
		}
	}
	return result
}

func getFormatableChangeloger(ctx *context.Context) (formatableChangeloger, error) {
	switch ctx.Config.Changelog.Use {
	case useGit, "":
		return gitChangeloger{}, nil
	case useGitLab, useGitea, useGitHub:
		if ctx.Git.PreviousTag == "" {
			log.Warnf("there's no previous tag, using 'git' instead of '%s'", ctx.Config.Changelog.Use)
			return gitChangeloger{}, nil
		}
		return newSCMChangeloger(ctx)
	default:
		return nil, fmt.Errorf("invalid changelog.use: %q", ctx.Config.Changelog.Use)
	}
}

func newGithubChangeloger(ctx *context.Context) (changeloger, error) {
	cli, err := client.NewGitHubReleaseNotesGenerator(ctx, ctx.Token)
	if err != nil {
		return nil, err
	}
	repo, err := git.ExtractRepoFromConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := repo.CheckSCM(); err != nil {
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

func newSCMChangeloger(ctx *context.Context) (formatableChangeloger, error) {
	cli, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	repo, err := git.ExtractRepoFromConfig(ctx)
	if err != nil {
		return nil, err
	}
	if err := repo.CheckSCM(); err != nil {
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
		log.Debugf("loading template %q", tmplName)
		content, err := loadFromFile(tmplName)
		if err != nil {
			return "", err
		}
		content, err = tmpl.New(ctx).Apply(content)
		if strings.TrimSpace(content) == "" && err == nil {
			log.Warnf("loaded %q, but it evaluates to an empty string", tmplName)
		}
		return content, err
	}

	if fileName != "" {
		log.Debugf("loading file %q", fileName)
		content, err := loadFromFile(fileName)
		if strings.TrimSpace(content) == "" && err == nil {
			log.Warnf("loaded %q, but it is empty", fileName)
		}
		return content, err
	}

	return "", nil
}

type formatableChangeloger interface {
	Log(ctx *context.Context) ([]client.ChangelogItem, error)
}

type changeloger interface {
	Log(ctx *context.Context) (string, error)
}

type gitChangeloger struct{}

func (g gitChangeloger) Log(ctx *context.Context) ([]client.ChangelogItem, error) {
	args := []string{
		"log",
		"--no-decorate",
		"--no-color",
		`--pretty=format:{"sha":"%H","message":"%s","author":"%an","email":"%aE"}`,
	}
	// if prev is empty, it means we don't have a previous tag, so we don't
	// pass any more args, which should everything.
	// if current is empty, it shouldn't matter, as it will then log
	// `{prev}..`, which should log everything from prev to HEAD.
	prev, current := ctx.Git.PreviousTag, ctx.Git.CurrentTag
	if prev != "" {
		args = append(args, fmt.Sprintf("%s..%s", prev, current))
	}
	out, err := git.Run(ctx, args...)
	if err != nil {
		return nil, err
	}
	var entries []client.ChangelogItem
	for _, s := range strings.Split(out, "\n") {
		if strings.TrimSpace(s) == "" {
			continue
		}
		var entry client.ChangelogItem
		if err := json.Unmarshal([]byte(s), &entry); err != nil {
			return nil, fmt.Errorf("changelog: invalid json: %s: %s", s, err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type scmChangeloger struct {
	client client.Client
	repo   client.Repo
	abbrev int
}

func (c *scmChangeloger) Log(ctx *context.Context) ([]client.ChangelogItem, error) {
	prev, current := ctx.Git.PreviousTag, ctx.Git.CurrentTag
	return c.client.Changelog(ctx, c.repo, prev, current)
}

type githubNativeChangeloger struct {
	client client.ReleaseNotesGenerator
	repo   client.Repo
}

func (c *githubNativeChangeloger) Log(ctx *context.Context) (string, error) {
	return c.client.GenerateReleaseNotes(ctx, c.repo, ctx.Git.PreviousTag, ctx.Git.CurrentTag)
}
