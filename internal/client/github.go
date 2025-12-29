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
	"github.com/google/go-github/v80/github"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
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
	//nolint:gosec
	base.(*http.Transport).TLSClientConfig = &tls.Config{
		InsecureSkipVerify: ctx.Config.GitHubURLs.SkipTLSVerify,
	}
	base.(*http.Transport).Proxy = http.ProxyFromEnvironment
	httpClient.Transport.(*oauth2.Transport).Base = base

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
		PreviousTagName: github.Ptr(prev),
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
		result, resp, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current, opts)
		if err != nil {
			return nil, err
		}
		for _, commit := range result.Commits {
			log = append(log, ChangelogItem{
				SHA:            commit.GetSHA(),
				Message:        strings.Split(commit.Commit.GetMessage(), "\n")[0],
				AuthorName:     commit.GetAuthor().GetName(),
				AuthorEmail:    commit.GetAuthor().GetEmail(),
				AuthorUsername: commit.GetAuthor().GetLogin(),
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return log, nil
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
		cmp.Or(head.Owner, base.Owner),
		cmp.Or(head.Name, base.Name),
		cmp.Or(head.Branch, base.Branch),
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
	pr, res, err := c.client.PullRequests.Create(
		ctx,
		base.Owner,
		base.Name,
		&github.NewPullRequest{
			Title: github.Ptr(title),
			Base:  github.Ptr(base.Branch),
			Head:  github.Ptr(headString(base, head)),
			Body:  github.Ptr(strings.Join([]string{tpl, prFooter}, "\n")),
			Draft: github.Ptr(draft),
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
	res, resp, err := c.client.Repositories.MergeUpstream(
		ctx,
		head.Owner,
		head.Name,
		&github.RepoMergeUpstreamRequest{
			Branch: github.Ptr(branch),
		},
	)
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
		Message: github.Ptr(message),
	}

	// When using a GitHub App token, omit the committer to get automatic signed commits
	// See: https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification#signature-verification-for-bots
	if !commitAuthor.UseGitHubAppToken {
		options.Committer = &github.CommitAuthor{
			Name:  github.Ptr(commitAuthor.Name),
			Email: github.Ptr(commitAuthor.Email),
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
		_, res, err := c.client.Repositories.GetBranch(ctx, repo.Owner, repo.Name, branch, 100)
		if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
			return fmt.Errorf("could not get branch %q: %w", branch, err)
		}

		if res.StatusCode == http.StatusNotFound {
			defRef, _, err := c.client.Git.GetRef(ctx, repo.Owner, repo.Name, "refs/heads/"+defBranch)
			if err != nil {
				return fmt.Errorf("could not get ref %q: %w", "refs/heads/"+defBranch, err)
			}

			if _, resp, err := c.client.Git.CreateRef(ctx, repo.Owner, repo.Name, github.CreateRef{
				Ref: "refs/heads/" + branch,
				SHA: defRef.Object.GetSHA(),
			}); err != nil {
				rerr := new(github.ErrorResponse)
				if !errors.As(err, &rerr) || rerr.Message != "Reference already exists" {
					return fmt.Errorf("could not create ref %q from %q: %w: %s", "refs/heads/"+branch, defRef.Object.GetSHA(), err, bodyOf(resp))
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

	if file != nil {
		options.SHA = file.SHA
	}
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
		Name:    github.Ptr(title),
		TagName: github.Ptr(ctx.Git.CurrentTag),
		Body:    github.Ptr(body),
		// Always start with a draft release while uploading artifacts.
		// PublishRelease will undraft it.
		Draft:      github.Ptr(true),
		Prerelease: github.Ptr(ctx.PreRelease),
	}

	if target := ctx.Config.Release.TargetCommitish; target != "" {
		target, err := tmpl.New(ctx).Apply(target)
		if err != nil {
			return "", err
		}
		if target != "" {
			data.TargetCommitish = github.Ptr(target)
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
	draft := ctx.Config.Release.Draft
	if draft {
		return nil
	}
	releaseIDInt, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return fmt.Errorf("non-numeric release ID %q: %w", releaseID, err)
	}
	data := &github.RepositoryRelease{
		Draft: github.Ptr(draft),
	}
	latest, err := tmpl.New(ctx).Apply(ctx.Config.Release.MakeLatest)
	if err != nil {
		return fmt.Errorf("templating GitHub make_latest: %w", err)
	}
	if latest != "" {
		data.MakeLatest = github.Ptr(latest)
	}
	if ctx.Config.Release.DiscussionCategoryName != "" {
		data.DiscussionCategoryName = github.Ptr(ctx.Config.Release.DiscussionCategoryName)
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
		release, resp, err := c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			data,
		)
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
	data.Body = github.Ptr(getReleaseNotes(release.GetBody(), body, ctx.Config.Release.ReleaseNotesMode))
	return c.updateRelease(ctx, release.GetID(), data)
}

func (c *githubClient) findRelease(ctx *context.Context, name string) (*github.RepositoryRelease, error) {
	if !ctx.Config.Release.UseExistingDraft {
		release, _, err := c.client.Repositories.GetReleaseByTag(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			name,
		)
		return release, err
	}
	return c.findDraftRelease(ctx, name)
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
	log.WithField("name", data.GetName()).
		WithField("release-id", release.GetID()).
		WithField("request-id", resp.Header.Get("X-GitHub-Request-Id")).
		Debug("release updated")
	return release, err
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

func (c *githubClient) deleteExistingDraftRelease(ctx *context.Context, name string) error {
	c.checkRateLimit(ctx)
	release, err := c.findDraftRelease(ctx, name)
	if err != nil {
		return fmt.Errorf("could not delete existing drafts: %w", err)
	}
	if release != nil {
		if _, err := c.client.Repositories.DeleteRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			release.GetID(),
		); err != nil {
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
		releases, resp, err := c.client.Repositories.ListReleases(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			&opt,
		)
		if err != nil {
			return nil, fmt.Errorf("could not list existing drafts: %w", err)
		}
		for _, r := range releases {
			if r.GetDraft() && r.GetName() == name {
				return r, nil
			}
		}
		if resp.NextPage == 0 {
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
	bts, _ := io.ReadAll(resp.Body)
	return string(bts)
}
