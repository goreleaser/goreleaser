package client

import (
	"cmp"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/google/go-github/v84/github"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/changelog"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"golang.org/x/oauth2"
)

const DefaultGitHubDownloadURL = "https://github.com"

var (
	_ Client                = &githubClient{}
	_ ReleaseNotesGenerator = &githubClient{}
	_ PullRequestOpener     = &githubClient{}
	_ ForkSyncer            = &githubClient{}
)

type githubClient struct {
	client *github.Client
}

// githubDo wraps a go-github SDK call with retry logic.
// It captures the response for status-code-based retry decisions.
func githubDo[T any](ctx *context.Context, fn func() (T, *github.Response, error)) (T, *github.Response, error) {
	var result T
	var resp *github.Response
	err := retryx.Do(ctx, ctx.Config.Retry, func() error {
		var err error
		result, resp, err = fn()
		if err != nil {
			return retryx.HTTP(err, must(resp).Response)
		}
		return nil
	}, retryx.IsRetriable)
	return result, resp, err
}

// NewGitHubReleaseNotesGenerator returns a GitHub client that can generate
// changelogs.
func NewGitHubReleaseNotesGenerator(ctx *context.Context, token string) (ReleaseNotesGenerator, error) {
	return newGitHub(ctx, token)
}

// newGitHub returns a github client implementation.
func newGitHub(ctx *context.Context, token string) (*githubClient, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	httpClient := oauth2.NewClient(ctx, ts)
	base := httpClient.Transport.(*oauth2.Transport).Base
	if base == nil || reflect.ValueOf(base).IsNil() {
		base = http.DefaultTransport
	}
	transport := base.(*http.Transport).Clone()
	//nolint:gosec
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.GitHubURLs.SkipTLSVerify,
	}
	transport.Proxy = http.ProxyFromEnvironment
	httpClient.Transport.(*oauth2.Transport).Base = transport

	baseURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.API)
	if err != nil {
		return nil, fmt.Errorf("templating GitHub API URL: %w", err)
	}
	uploadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.Upload)
	if err != nil {
		return nil, fmt.Errorf("templating GitHub upload URL: %w", err)
	}

	if baseURL == "" {
		return &githubClient{client: github.NewClient(httpClient)}, nil
	}

	client, err := github.NewClient(httpClient).WithEnterpriseURLs(baseURL, uploadURL)
	if err != nil {
		return &githubClient{}, err
	}
	return &githubClient{client: client}, nil
}

func (c *githubClient) checkRateLimit(ctx *context.Context) {
	c.rateLimitChecker(ctx, 100, func(limits *github.RateLimits) *github.Rate {
		return limits.Core
	})
}

func (c *githubClient) rateLimitChecker(
	ctx *context.Context,
	target int,
	which func(*github.RateLimits) *github.Rate,
) {
	limits, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		log.Warn("could not check rate limits, hoping for the best...")
		return
	}
	rate := which(limits)
	if rate.Remaining > target {
		return
	}
	// sometimes, after the rate limit just reset, it might still report
	// low remaining and a reset time in the past - sleep at least 5s
	sleep := max(time.Until(rate.Reset.Time), 5*time.Second)
	log.Warnf("rate limit almost reached (%d remaining), sleeping for %s...", rate.Remaining, sleep)
	select {
	case <-time.After(sleep):
	case <-ctx.Done():
		log.Warnf("context cancelled while waiting for rate limit to reset: %v", ctx.Err())
	}
}

func (c *githubClient) GenerateReleaseNotes(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	c.checkRateLimit(ctx)
	notes, _, err := githubDo(ctx, func() (*github.RepositoryReleaseNotes, *github.Response, error) {
		return c.client.Repositories.GenerateReleaseNotes(ctx, repo.Owner, repo.Name, &github.GenerateNotesOptions{
			TagName:         current,
			PreviousTagName: &prev,
		})
	})
	if err != nil {
		return "", err
	}
	return notes.Body, err
}

func (c *githubClient) Changelog(ctx *context.Context, repo Repo, prev, current string) ([]ChangelogItem, error) {
	c.checkRateLimit(ctx)
	var log []ChangelogItem
	opts := &github.ListOptions{PerPage: 100}

	for {
		result, resp, err := githubDo(ctx, func() (*github.CommitsComparison, *github.Response, error) {
			return c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current, opts)
		})
		if err != nil {
			return nil, err
		}
		for _, commit := range result.Commits {
			var authors []Author
			if author := commit.GetAuthor(); author != nil {
				authors = append(authors, Author{
					Name:     author.GetName(),
					Email:    author.GetEmail(),
					Username: author.GetLogin(),
				})
			}
			coauthors := changelog.ExtractCoAuthors(commit.Commit.GetMessage())
			authors = append(authors, c.authorsLookup(coauthors)...)
			log = append(log, fillDeprecated(ChangelogItem{
				SHA:     commit.GetSHA(),
				Message: strings.Split(commit.Commit.GetMessage(), "\n")[0],
				Authors: authors,
			}))
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return log, nil
}

func (c *githubClient) authorsLookup(authors []Author) []Author {
	for i := range authors {
		author := &authors[i]
		before, ok := strings.CutSuffix(author.Email, "@users.noreply.github.com")
		if !ok {
			continue
		}
		// GitHub noreply format: ID+USERNAME@users.noreply.github.com
		if _, clean, ok := strings.Cut(before, "+"); ok {
			author.Username = clean
			continue
		}
		author.Username = before
	}
	return authors
}

// getDefaultBranch returns the default branch of a github repo
func (c *githubClient) getDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	c.checkRateLimit(ctx)
	p, res, err := githubDo(ctx, func() (*github.Repository, *github.Response, error) {
		return c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
	})
	if err != nil {
		log := log.WithField("projectID", repo.String())
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.
			WithError(err).
			Warn("error checking for default branch")
		return "", err
	}
	return p.GetDefaultBranch(), nil
}

// CloseMilestone closes a given milestone.
func (c *githubClient) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	c.checkRateLimit(ctx)
	milestone, err := c.getMilestoneByTitle(ctx, repo, title)
	if err != nil {
		return err
	}

	if milestone == nil {
		return ErrNoMilestoneFound{Title: title}
	}

	closedState := "closed"
	milestone.State = &closedState

	_, _, err = githubDo(ctx, func() (*github.Milestone, *github.Response, error) {
		return c.client.Issues.EditMilestone(
			ctx,
			repo.Owner,
			repo.Name,
			*milestone.Number,
			milestone,
		)
	})

	return err
}

func headString(base, head Repo) string {
	return strings.Join([]string{
		cmp.Or(head.Owner, base.Owner),
		cmp.Or(head.Name, base.Name),
		cmp.Or(head.Branch, base.Branch),
	}, ":")
}

func (c *githubClient) getPRTemplate(ctx *context.Context, repo Repo) (string, error) {
	content, _, err := githubDo(ctx, func() (*github.RepositoryContent, *github.Response, error) {
		content, _, resp, err := c.client.Repositories.GetContents(
			ctx, repo.Owner, repo.Name,
			".github/PULL_REQUEST_TEMPLATE.md",
			&github.RepositoryContentGetOptions{
				Ref: repo.Branch,
			},
		)
		return content, resp, err
	})
	if err != nil {
		return "", err
	}
	return content.GetContent()
}

func (c *githubClient) OpenPullRequest(
	ctx *context.Context,
	base, head Repo,
	title string,
	draft bool,
) error {
	c.checkRateLimit(ctx)
	base.Owner = cmp.Or(base.Owner, head.Owner)
	base.Name = cmp.Or(base.Name, head.Name)
	if base.Branch == "" {
		def, err := c.getDefaultBranch(ctx, base)
		if err != nil {
			return err
		}
		base.Branch = def
	}
	tpl, err := c.getPRTemplate(ctx, base)
	if err != nil {
		log.WithError(err).Debug("no pull request template found...")
	}
	if len(tpl) > 0 {
		log.Info("got a pr template")
	}

	log := log.
		WithField("base", headString(base, Repo{})).
		WithField("head", headString(base, head)).
		WithField("draft", draft)
	log.Info("opening pull request")
	pr, res, err := githubDo(ctx, func() (*github.PullRequest, *github.Response, error) {
		return c.client.PullRequests.Create(
			ctx,
			base.Owner,
			base.Name,
			&github.NewPullRequest{
				Title: &title,
				Base:  &base.Branch,
				Head:  new(headString(base, head)),
				Body:  new(strings.Join([]string{tpl, prFooter}, "\n")),
				Draft: &draft,
			},
		)
	})
	if err != nil {
		if res != nil && res.StatusCode == http.StatusUnprocessableEntity {
			log.WithError(err).Warn("pull request validation failed")
			return nil
		}
		return fmt.Errorf("could not create pull request: %w", err)
	}
	log.WithField("url", pr.GetHTMLURL()).Info("pull request created")
	return nil
}

func (c *githubClient) SyncFork(ctx *context.Context, head, base Repo) error {
	branch := base.Branch
	if branch == "" {
		def, err := c.getDefaultBranch(ctx, base)
		if err != nil {
			return err
		}
		branch = def
	}
	res, resp, err := githubDo(ctx, func() (*github.RepoMergeUpstreamResult, *github.Response, error) {
		return c.client.Repositories.MergeUpstream(
			ctx,
			head.Owner,
			head.Name,
			&github.RepoMergeUpstreamRequest{
				Branch: &branch,
			},
		)
	})
	if err != nil {
		return fmt.Errorf("%w: %s", err, bodyOf(resp))
	}
	log.WithField("merge_type", res.GetMergeType()).
		WithField("base_branch", res.GetBaseBranch()).
		Info(res.GetMessage())
	return nil
}

func (c *githubClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	content []byte,
	path,
	message string,
) error {
	c.checkRateLimit(ctx)
	defBranch, err := c.getDefaultBranch(ctx, repo)
	if err != nil {
		return fmt.Errorf("could not get default branch: %w", err)
	}

	branch := repo.Branch
	if branch == "" {
		branch = defBranch
	}

	options := &github.RepositoryContentFileOptions{
		Content: content,
		Message: &message,
	}

	// When using a GitHub App token, omit the committer to get automatic signed commits
	// See: https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification#signature-verification-for-bots
	if !commitAuthor.UseGitHubAppToken {
		options.Committer = &github.CommitAuthor{
			Name:  &commitAuthor.Name,
			Email: &commitAuthor.Email,
		}
	}

	// Set the branch if we got it above...otherwise, just default to
	// whatever the SDK does auto-magically
	if branch != "" {
		options.Branch = &branch
	}

	log.
		WithField("repository", repo.String()).
		WithField("branch", repo.Branch).
		WithField("file", path).
		Info("pushing")

	if defBranch != branch && branch != "" {
		_, res, err := githubDo(ctx, func() (*github.Branch, *github.Response, error) {
			return c.client.Repositories.GetBranch(ctx, repo.Owner, repo.Name, branch, 100)
		})
		if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
			return fmt.Errorf("could not get branch %q: %w", branch, err)
		}

		if res != nil && res.StatusCode == http.StatusNotFound {
			defRef, _, err := githubDo(ctx, func() (*github.Reference, *github.Response, error) {
				return c.client.Git.GetRef(ctx, repo.Owner, repo.Name, "refs/heads/"+defBranch)
			})
			if err != nil {
				return fmt.Errorf("could not get ref %q: %w", "refs/heads/"+defBranch, err)
			}

			_, resp, err := githubDo(ctx, func() (*github.Reference, *github.Response, error) {
				return c.client.Git.CreateRef(ctx, repo.Owner, repo.Name, github.CreateRef{
					Ref: "refs/heads/" + branch,
					SHA: defRef.Object.GetSHA(),
				})
			})
			if err != nil {
				rerr := new(github.ErrorResponse)
				if !errors.As(err, &rerr) || rerr.Message != "Reference already exists" {
					return fmt.Errorf("could not create ref %q from %q: %w: %s", "refs/heads/"+branch, defRef.Object.GetSHA(), err, bodyOf(resp))
				}
			}
		}
	}

	file, res, err := githubDo(ctx, func() (*github.RepositoryContent, *github.Response, error) {
		content, _, r, err := c.client.Repositories.GetContents(
			ctx,
			repo.Owner,
			repo.Name,
			path,
			&github.RepositoryContentGetOptions{
				Ref: branch,
			},
		)
		return content, r, err
	})
	if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
		return fmt.Errorf("could not get %q: %w", path, err)
	}

	if file != nil {
		options.SHA = file.SHA
	}
	if _, _, err := githubDo(ctx, func() (*github.RepositoryContentResponse, *github.Response, error) {
		return c.client.Repositories.UpdateFile(
			ctx,
			repo.Owner,
			repo.Name,
			path,
			options,
		)
	}); err != nil {
		return fmt.Errorf("could not update %q: %w", path, err)
	}
	return nil
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	tpl := tmpl.New(ctx)
	title, err := tpl.Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}
	c.checkRateLimit(ctx)

	if ctx.Config.Release.Draft && ctx.Config.Release.ReplaceExistingDraft {
		if err := c.deleteExistingDraftRelease(ctx, title); err != nil {
			return "", err
		}
	}

	// Truncate the release notes if it's too long (github doesn't allow more than 125000 characters)
	body = truncateReleaseBody(body)

	data := &github.RepositoryRelease{
		Name:    &title,
		TagName: &ctx.Git.CurrentTag,
		Body:    &body,
		// Always start with a draft release while uploading artifacts.
		// PublishRelease will undraft it.
		Draft:      new(true),
		Prerelease: &ctx.PreRelease,
	}

	if target := ctx.Config.Release.TargetCommitish; target != "" {
		target, err := tpl.Apply(target)
		if err != nil {
			return "", err
		}
		if target != "" {
			data.TargetCommitish = &target
		}
	}

	release, err := c.createOrUpdateRelease(ctx, data, body)
	if err != nil {
		return "", fmt.Errorf("could not release: %w", err)
	}

	log.WithField("url", release.GetHTMLURL()).Info("created")
	return strconv.FormatInt(release.GetID(), 10), nil
}

func (c *githubClient) PublishRelease(ctx *context.Context, releaseID string) error {
	if ctx.Config.Release.Draft {
		return nil
	}
	releaseIDInt, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return fmt.Errorf("non-numeric release ID %q: %w", releaseID, err)
	}
	data := &github.RepositoryRelease{
		Draft: new(false),
	}
	tpl := tmpl.New(ctx)
	title, err := tpl.Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return fmt.Errorf("templating GitHub release name: %w", err)
	}
	if title != "" {
		data.Name = &title
	}
	if ctx.PreRelease {
		data.Prerelease = &ctx.PreRelease
	}
	latest, err := tpl.Apply(ctx.Config.Release.MakeLatest)
	if err != nil {
		return fmt.Errorf("templating GitHub make_latest: %w", err)
	}
	if ctx.PreRelease {
		latest = "false"
	}
	if latest != "" {
		data.MakeLatest = &latest
	}
	if ctx.Config.Release.DiscussionCategoryName != "" {
		data.DiscussionCategoryName = &ctx.Config.Release.DiscussionCategoryName
	}
	release, err := c.updateRelease(ctx, releaseIDInt, data)
	if err != nil {
		return fmt.Errorf("could not update existing release: %w", err)
	}
	log.WithField("url", release.GetHTMLURL()).Debug("published")
	return nil
}

func (c *githubClient) createOrUpdateRelease(ctx *context.Context, data *github.RepositoryRelease, body string) (*github.RepositoryRelease, error) {
	c.checkRateLimit(ctx)
	release, err := c.findRelease(ctx, data.GetTagName())
	if err != nil || release == nil {
		release, resp, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
			return c.client.Repositories.CreateRelease(
				ctx,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
				data,
			)
		})
		if resp == nil {
			log.WithField("name", data.GetName()).
				WithError(err).
				Debug("release creation failed")
			return nil, err
		}
		if err != nil {
			log.WithField("name", data.GetName()).
				WithField("request-id", resp.Header.Get("X-Github-Request-Id")).
				WithError(err).
				Debug("release creation failed")
			return nil, err
		}
		log.WithField("name", data.GetName()).
			WithField("release-id", release.GetID()).
			WithField("request-id", resp.Header.Get("X-GitHub-Request-Id")).
			Debug("release created")
		return release, err
	}

	data.Draft = release.Draft
	data.Body = new(getReleaseNotes(release.GetBody(), body, ctx.Config.Release.ReleaseNotesMode))
	return c.updateRelease(ctx, release.GetID(), data)
}

func (c *githubClient) findRelease(ctx *context.Context, name string) (*github.RepositoryRelease, error) {
	if !ctx.Config.Release.UseExistingDraft {
		release, _, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
			return c.client.Repositories.GetReleaseByTag(
				ctx,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
				name,
			)
		})
		return release, err
	}
	return c.findDraftRelease(ctx, name)
}

func (c *githubClient) updateRelease(ctx *context.Context, id int64, data *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	c.checkRateLimit(ctx)
	release, resp, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
		return c.client.Repositories.EditRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			id,
			data,
		)
	})
	if err != nil {
		return nil, err
	}
	l := log.WithField("name", data.GetName()).
		WithField("release-id", release.GetID())
	if resp != nil {
		l = l.WithField("request-id", resp.Header.Get("X-GitHub-Request-Id"))
	}
	l.Debug("release updated")
	return release, nil
}

func (c *githubClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	downloadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.Download)
	if err != nil {
		return "", fmt.Errorf("templating GitHub download URL: %w", err)
	}

	return fmt.Sprintf(
		"%s/%s/%s/releases/download/{{ urlPathEscape .Tag }}/{{ .ArtifactName }}",
		downloadURL,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
	), nil
}

func (c *githubClient) deleteReleaseArtifact(ctx *context.Context, releaseID int64, name string, page int) error {
	c.checkRateLimit(ctx)
	log.WithField("name", name).Info("delete pre-existing asset from the release")
	assets, resp, err := githubDo(ctx, func() ([]*github.ReleaseAsset, *github.Response, error) {
		return c.client.Repositories.ListReleaseAssets(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			releaseID,
			&github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		)
	})
	if err != nil {
		githubErrLogger(resp, err).
			WithField("release-id", releaseID).
			Warn("could not list release assets")
		return err
	}
	for _, asset := range assets {
		if asset.GetName() != name {
			continue
		}
		resp, err := retryx.DoWithData(ctx, ctx.Config.Retry, func() (*github.Response, error) {
			r, err := c.client.Repositories.DeleteReleaseAsset(
				ctx,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
				asset.GetID(),
			)
			if err != nil {
				return r, retryx.HTTP(err, must(r).Response)
			}
			return r, nil
		}, retryx.IsRetriable)
		if err != nil {
			githubErrLogger(resp, err).
				WithField("release-id", releaseID).
				WithField("id", asset.GetID()).
				WithField("name", name).
				Warn("could not delete asset")
		}
		return err
	}
	if resp != nil {
		if next := resp.NextPage; next > 0 {
			return c.deleteReleaseArtifact(ctx, releaseID, name, next)
		}
	}
	return nil
}

func (c *githubClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
) error {
	c.checkRateLimit(ctx)
	githubReleaseID, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return err
	}

	return retryx.Do(ctx, ctx.Config.Retry, func() error {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		defer file.Close()

		_, resp, err := c.client.Repositories.UploadReleaseAsset(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			githubReleaseID,
			&github.UploadOptions{
				Name: artifact.Name,
			},
			file,
		)
		if err == nil {
			return nil
		}

		githubErrLogger(resp, err).
			WithField("name", artifact.Name).
			WithField("release-id", releaseID).
			Warn("upload failed")

		// this status means the asset already exists
		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			if !ctx.Config.Release.ReplaceExistingArtifacts {
				return retryx.Unrecoverable(err)
			}
			// if the user allowed to delete assets, we delete it, and return
			// a retriable error so we try again.
			if delErr := c.deleteReleaseArtifact(ctx, githubReleaseID, artifact.Name, 1); delErr != nil {
				return retryx.Unrecoverable(delErr)
			}
			return retryx.Retriable(err)
		}
		return retryx.HTTP(err, must(resp).Response)
	}, retryx.IsRetriable)
}

// getMilestoneByTitle returns a milestone by title.
func (c *githubClient) getMilestoneByTitle(ctx *context.Context, repo Repo, title string) (*github.Milestone, error) {
	c.checkRateLimit(ctx)
	// The GitHub API/SDK does not provide lookup by title functionality currently.
	opts := &github.MilestoneListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		milestones, resp, err := githubDo(ctx, func() ([]*github.Milestone, *github.Response, error) {
			return c.client.Issues.ListMilestones(
				ctx,
				repo.Owner,
				repo.Name,
				opts,
			)
		})
		if err != nil {
			return nil, err
		}

		for _, m := range milestones {
			if m != nil && m.Title != nil && *m.Title == title {
				return m, nil
			}
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return nil, nil
}

func (c *githubClient) deleteExistingDraftRelease(ctx *context.Context, name string) error {
	c.checkRateLimit(ctx)
	release, err := c.findDraftRelease(ctx, name)
	if err != nil {
		return fmt.Errorf("could not delete existing drafts: %w", err)
	}
	if release != nil {
		err := retryx.Do(ctx, ctx.Config.Retry, func() error {
			_, err := c.client.Repositories.DeleteRelease(
				ctx,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
				release.GetID(),
			)
			return err
		}, retryx.IsNetworkError)
		if err != nil {
			return fmt.Errorf("could not delete previous draft release: %w", err)
		}

		log.WithField("commit", release.GetTargetCommitish()).
			WithField("tag", release.GetTagName()).
			WithField("name", release.GetName()).
			Info("deleted previous draft release")
	}
	return nil
}

func (c *githubClient) findDraftRelease(ctx *context.Context, name string) (*github.RepositoryRelease, error) {
	c.checkRateLimit(ctx)
	opt := github.ListOptions{PerPage: 50}
	for {
		releases, resp, err := githubDo(ctx, func() ([]*github.RepositoryRelease, *github.Response, error) {
			return c.client.Repositories.ListReleases(
				ctx,
				ctx.Config.Release.GitHub.Owner,
				ctx.Config.Release.GitHub.Name,
				&opt,
			)
		})
		if err != nil {
			return nil, fmt.Errorf("could not list existing drafts: %w", err)
		}
		for _, r := range releases {
			if r.GetDraft() && r.GetName() == name {
				return r, nil
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return nil, nil
		}
		opt.Page = resp.NextPage
	}
}

func githubErrLogger(resp *github.Response, err error) *log.Entry {
	requestID := ""
	if resp != nil {
		requestID = resp.Header.Get("X-GitHub-Request-Id")
	}
	return log.WithField("request-id", requestID).WithError(err)
}

func bodyOf(resp *github.Response) string {
	if resp == nil || resp.Body == nil {
		return "no response"
	}
	defer resp.Body.Close()
	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("could not read response body: %v", err)
	}
	return string(bts)
}
