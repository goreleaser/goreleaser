package client

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/log"
	"github.com/charmbracelet/x/exp/ordered"
	"github.com/google/go-github/v61/github"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
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
	// nolint: gosec
	base.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.GitHubURLs.SkipTLSVerify,
	}
	base.(*http.Transport).Proxy = http.ProxyFromEnvironment
	httpClient.Transport.(*oauth2.Transport).Base = base

	client := github.NewClient(httpClient)
	err := overrideGitHubClientAPI(ctx, client)
	if err != nil {
		return &githubClient{}, err
	}

	return &githubClient{client: client}, nil
}

func (c *githubClient) checkRateLimit(ctx *context.Context) {
	limits, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		log.Warn("could not check rate limits, hoping for the best...")
		return
	}
	if limits.Core.Remaining > 100 { // 100 should be safe enough
		return
	}
	sleep := limits.Core.Reset.UTC().Sub(time.Now().UTC())
	if sleep <= 0 {
		// it seems that sometimes, after the rate limit just reset, it might
		// still get <100 remaining and a reset time in the past... in such
		// cases we can probably sleep a bit more before trying again...
		sleep = 15 * time.Second
	}
	log.Warnf("token too close to rate limiting, will sleep for %s before continuing...", sleep)
	time.Sleep(sleep)
	c.checkRateLimit(ctx)
}

func (c *githubClient) GenerateReleaseNotes(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	c.checkRateLimit(ctx)
	notes, _, err := c.client.Repositories.GenerateReleaseNotes(ctx, repo.Owner, repo.Name, &github.GenerateNotesOptions{
		TagName:         current,
		PreviousTagName: github.String(prev),
	})
	if err != nil {
		return "", err
	}
	return notes.Body, err
}

func (c *githubClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	c.checkRateLimit(ctx)
	var log []string
	opts := &github.ListOptions{PerPage: 100}

	for {
		result, resp, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current, opts)
		if err != nil {
			return "", err
		}
		for _, commit := range result.Commits {
			log = append(log, fmt.Sprintf(
				"%s: %s (@%s)",
				commit.GetSHA(),
				strings.Split(commit.Commit.GetMessage(), "\n")[0],
				commit.GetAuthor().GetLogin(),
			))
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return strings.Join(log, "\n"), nil
}

// getDefaultBranch returns the default branch of a github repo
func (c *githubClient) getDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	c.checkRateLimit(ctx)
	p, res, err := c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
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

	_, _, err = c.client.Issues.EditMilestone(
		ctx,
		repo.Owner,
		repo.Name,
		*milestone.Number,
		milestone,
	)

	return err
}

func headString(base, head Repo) string {
	return strings.Join([]string{
		ordered.First(head.Owner, base.Owner),
		ordered.First(head.Name, base.Name),
		ordered.First(head.Branch, base.Branch),
	}, ":")
}

func (c *githubClient) getPRTemplate(ctx *context.Context, repo Repo) (string, error) {
	content, _, _, err := c.client.Repositories.GetContents(
		ctx, repo.Owner, repo.Name,
		".github/PULL_REQUEST_TEMPLATE.md",
		&github.RepositoryContentGetOptions{
			Ref: repo.Branch,
		},
	)
	if err != nil {
		return "", err
	}
	return content.GetContent()
}

const prFooter = "###### Automated with [GoReleaser](https://goreleaser.com)"

func (c *githubClient) OpenPullRequest(
	ctx *context.Context,
	base, head Repo,
	title string,
	draft bool,
) error {
	c.checkRateLimit(ctx)
	base.Owner = ordered.First(base.Owner, head.Owner)
	base.Name = ordered.First(base.Name, head.Name)
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
	pr, res, err := c.client.PullRequests.Create(
		ctx,
		base.Owner,
		base.Name,
		&github.NewPullRequest{
			Title: github.String(title),
			Base:  github.String(base.Branch),
			Head:  github.String(headString(base, head)),
			Body:  github.String(strings.Join([]string{tpl, prFooter}, "\n")),
			Draft: github.Bool(draft),
		},
	)
	if err != nil {
		if res.StatusCode == http.StatusUnprocessableEntity {
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
	res, _, err := c.client.Repositories.MergeUpstream(
		ctx,
		head.Owner,
		head.Name,
		&github.RepoMergeUpstreamRequest{
			Branch: github.String(branch),
		},
	)
	if res != nil {
		log.WithField("merge_type", res.GetMergeType()).
			WithField("base_branch", res.GetBaseBranch()).
			Info(res.GetMessage())
	}
	return err
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
		Committer: &github.CommitAuthor{
			Name:  github.String(commitAuthor.Name),
			Email: github.String(commitAuthor.Email),
		},
		Content: content,
		Message: github.String(message),
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
		_, res, err := c.client.Repositories.GetBranch(ctx, repo.Owner, repo.Name, branch, 100)
		if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
			return fmt.Errorf("could not get branch %q: %w", branch, err)
		}

		if res.StatusCode == http.StatusNotFound {
			defRef, _, err := c.client.Git.GetRef(ctx, repo.Owner, repo.Name, "refs/heads/"+defBranch)
			if err != nil {
				return fmt.Errorf("could not get ref %q: %w", "refs/heads/"+defBranch, err)
			}

			if _, _, err := c.client.Git.CreateRef(ctx, repo.Owner, repo.Name, &github.Reference{
				Ref: github.String("refs/heads/" + branch),
				Object: &github.GitObject{
					SHA: defRef.Object.SHA,
				},
			}); err != nil {
				rerr := new(github.ErrorResponse)
				if !errors.As(err, &rerr) || rerr.Message != "Reference already exists" {
					return fmt.Errorf("could not create ref %q from %q: %w", "refs/heads/"+branch, defRef.Object.GetSHA(), err)
				}
			}
		}
	}

	file, _, res, err := c.client.Repositories.GetContents(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		&github.RepositoryContentGetOptions{
			Ref: branch,
		},
	)
	if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
		return fmt.Errorf("could not get %q: %w", path, err)
	}

	options.SHA = github.String(file.GetSHA())
	if _, _, err := c.client.Repositories.UpdateFile(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		options,
	); err != nil {
		return fmt.Errorf("could not update %q: %w", path, err)
	}
	return nil
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	c.checkRateLimit(ctx)
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}

	if ctx.Config.Release.Draft && ctx.Config.Release.ReplaceExistingDraft {
		if err := c.deleteExistingDraftRelease(ctx, title); err != nil {
			return "", err
		}
	}

	// Truncate the release notes if it's too long (github doesn't allow more than 125000 characters)
	body = truncateReleaseBody(body)

	data := &github.RepositoryRelease{
		Name:    github.String(title),
		TagName: github.String(ctx.Git.CurrentTag),
		Body:    github.String(body),
		// Always start with a draft release while uploading artifacts.
		// PublishRelease will undraft it.
		Draft:      github.Bool(true),
		Prerelease: github.Bool(ctx.PreRelease),
		MakeLatest: github.String("true"),
	}

	if ctx.Config.Release.DiscussionCategoryName != "" {
		data.DiscussionCategoryName = github.String(ctx.Config.Release.DiscussionCategoryName)
	}

	if target := ctx.Config.Release.TargetCommitish; target != "" {
		target, err := tmpl.New(ctx).Apply(target)
		if err != nil {
			return "", err
		}
		if target != "" {
			data.TargetCommitish = github.String(target)
		}
	}

	if latest := strings.TrimSpace(ctx.Config.Release.MakeLatest); latest == "false" {
		data.MakeLatest = github.String(latest)
	}

	release, err := c.createOrUpdateRelease(ctx, data, body)
	if err != nil {
		return "", fmt.Errorf("could not release: %w", err)
	}

	return strconv.FormatInt(release.GetID(), 10), nil
}

func (c *githubClient) PublishRelease(ctx *context.Context, releaseID string) error {
	draft := ctx.Config.Release.Draft
	if draft {
		return nil
	}
	releaseIDInt, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return fmt.Errorf("non-numeric release ID %q: %w", releaseID, err)
	}
	if _, err := c.updateRelease(ctx, releaseIDInt, &github.RepositoryRelease{
		Draft: github.Bool(draft),
	}); err != nil {
		return fmt.Errorf("could not update existing release: %w", err)
	}
	return nil
}

func (c *githubClient) createOrUpdateRelease(ctx *context.Context, data *github.RepositoryRelease, body string) (*github.RepositoryRelease, error) {
	c.checkRateLimit(ctx)
	release, _, err := c.client.Repositories.GetReleaseByTag(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		data.GetTagName(),
	)
	if err != nil {
		release, resp, err := c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			data,
		)
		if err == nil {
			log.WithField("name", data.GetName()).
				WithField("release-id", release.GetID()).
				WithField("request-id", resp.Header.Get("X-GitHub-Request-Id")).
				Info("release created")
		}
		return release, err
	}

	data.Draft = release.Draft
	data.Body = github.String(getReleaseNotes(release.GetBody(), body, ctx.Config.Release.ReleaseNotesMode))
	return c.updateRelease(ctx, release.GetID(), data)
}

func (c *githubClient) updateRelease(ctx *context.Context, id int64, data *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	c.checkRateLimit(ctx)
	release, resp, err := c.client.Repositories.EditRelease(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		id,
		data,
	)
	if err == nil {
		log.WithField("name", data.GetName()).
			WithField("release-id", release.GetID()).
			WithField("request-id", resp.Header.Get("X-GitHub-Request-Id")).
			Info("release updated")
	}
	return release, err
}

func (c *githubClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	downloadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.Download)
	if err != nil {
		return "", fmt.Errorf("templating GitHub download URL: %w", err)
	}

	return fmt.Sprintf(
		"%s/%s/%s/releases/download/{{ .Tag }}/{{ .ArtifactName }}",
		downloadURL,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
	), nil
}

func (c *githubClient) deleteReleaseArtifact(ctx *context.Context, releaseID int64, name string, page int) error {
	c.checkRateLimit(ctx)
	log.WithField("name", name).Info("delete pre-existing asset from the release")
	assets, resp, err := c.client.Repositories.ListReleaseAssets(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		releaseID,
		&github.ListOptions{
			PerPage: 100,
			Page:    page,
		},
	)
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
		resp, err := c.client.Repositories.DeleteReleaseAsset(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			asset.GetID(),
		)
		if err != nil {
			githubErrLogger(resp, err).
				WithField("release-id", releaseID).
				WithField("id", asset.GetID()).
				WithField("name", name).
				Warn("could not delete asset")
		}
		return err
	}
	if next := resp.NextPage; next > 0 {
		return c.deleteReleaseArtifact(ctx, releaseID, name, next)
	}
	return nil
}

func (c *githubClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
	file *os.File,
) error {
	c.checkRateLimit(ctx)
	githubReleaseID, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return err
	}
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
	if err != nil {
		githubErrLogger(resp, err).
			WithField("name", artifact.Name).
			WithField("release-id", releaseID).
			Warn("upload failed")
	}
	if err == nil {
		return nil
	}
	// this status means the asset already exists
	if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
		if !ctx.Config.Release.ReplaceExistingArtifacts {
			return err
		}
		// if the user allowed to delete assets, we delete it, and return a
		// retriable error.
		if err := c.deleteReleaseArtifact(ctx, githubReleaseID, artifact.Name, 1); err != nil {
			return err
		}
		return RetriableError{err}
	}
	return RetriableError{err}
}

// getMilestoneByTitle returns a milestone by title.
func (c *githubClient) getMilestoneByTitle(ctx *context.Context, repo Repo, title string) (*github.Milestone, error) {
	c.checkRateLimit(ctx)
	// The GitHub API/SDK does not provide lookup by title functionality currently.
	opts := &github.MilestoneListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		milestones, resp, err := c.client.Issues.ListMilestones(
			ctx,
			repo.Owner,
			repo.Name,
			opts,
		)
		if err != nil {
			return nil, err
		}

		for _, m := range milestones {
			if m != nil && m.Title != nil && *m.Title == title {
				return m, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return nil, nil
}

func overrideGitHubClientAPI(ctx *context.Context, client *github.Client) error {
	if ctx.Config.GitHubURLs.API == "" {
		return nil
	}

	apiURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.API)
	if err != nil {
		return fmt.Errorf("templating GitHub API URL: %w", err)
	}
	api, err := url.Parse(apiURL)
	if err != nil {
		return err
	}

	uploadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitHubURLs.Upload)
	if err != nil {
		return fmt.Errorf("templating GitHub upload URL: %w", err)
	}
	upload, err := url.Parse(uploadURL)
	if err != nil {
		return err
	}

	client.BaseURL = api
	client.UploadURL = upload

	return nil
}

func (c *githubClient) deleteExistingDraftRelease(ctx *context.Context, name string) error {
	c.checkRateLimit(ctx)
	opt := github.ListOptions{PerPage: 50}
	for {
		releases, resp, err := c.client.Repositories.ListReleases(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			&opt,
		)
		if err != nil {
			return fmt.Errorf("could not delete existing drafts: %w", err)
		}
		for _, r := range releases {
			if r.GetDraft() && r.GetName() == name {
				if _, err := c.client.Repositories.DeleteRelease(
					ctx,
					ctx.Config.Release.GitHub.Owner,
					ctx.Config.Release.GitHub.Name,
					r.GetID(),
				); err != nil {
					return fmt.Errorf("could not delete previous draft release: %w", err)
				}

				log.WithField("commit", r.GetTargetCommitish()).
					WithField("tag", r.GetTagName()).
					WithField("name", r.GetName()).
					Info("deleted previous draft release")

				// in theory, there should be only 1 release matching, so we can just return
				return nil
			}
		}
		if resp.NextPage == 0 {
			return nil
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
