package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/apex/log"
	"github.com/google/go-github/v35/github"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/oauth2"
)

const DefaultGitHubDownloadURL = "https://github.com"

type githubClient struct {
	client *github.Client
}

// NewUnauthenticatedGitHub returns a github client that is not authenticated.
// Used in tests only.
func NewUnauthenticatedGitHub() Client {
	return &githubClient{client: github.NewClient(nil)}
}

// NewGitHub returns a github client implementation.
func NewGitHub(ctx *context.Context, token string) (Client, error) {
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

func (c *githubClient) Changelog(ctx *context.Context, repo Repo, prev, current string) (string, error) {
	result, _, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current)
	if err != nil {
		return "", err
	}
	var log []string
	for _, commit := range result.Commits {
		log = append(log, fmt.Sprintf(
			"%s: %s (@%s)",
			commit.GetSHA(),
			strings.Split(commit.Commit.GetMessage(), "\n")[0],
			commit.GetAuthor().GetLogin(),
		))
	}
	return strings.Join(log, "\n"), nil
}

// GetDefaultBranch returns the default branch of a github repo
func (c *githubClient) GetDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	p, res, err := c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID":  repo.String(),
			"statusCode": res.StatusCode,
			"err":        err.Error(),
		}).Warn("error checking for default branch")
		return "", err
	}
	return p.GetDefaultBranch(), nil
}

// CloseMilestone closes a given milestone.
func (c *githubClient) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
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

func (c *githubClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	content []byte,
	path,
	message string,
) error {
	var branch string
	var err error
	if repo.Branch != "" {
		branch = repo.Branch
	} else {
		branch, err = c.GetDefaultBranch(ctx, repo)
		if err != nil {
			// Fall back to sdk default
			log.WithFields(log.Fields{
				"fileName":        path,
				"projectID":       repo.String(),
				"requestedBranch": branch,
				"err":             err.Error(),
			}).Warn("error checking for default branch, using master")
		}
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

	file, _, res, err := c.client.Repositories.GetContents(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		&github.RepositoryContentGetOptions{},
	)
	if err != nil && (res == nil || res.StatusCode != 404) {
		return err
	}

	if res.StatusCode == 404 {
		_, _, err = c.client.Repositories.CreateFile(
			ctx,
			repo.Owner,
			repo.Name,
			path,
			options,
		)
		return err
	}
	options.SHA = file.SHA
	_, _, err = c.client.Repositories.UpdateFile(
		ctx,
		repo.Owner,
		repo.Name,
		path,
		options,
	)
	return err
}

func (c *githubClient) CreateRelease(ctx *context.Context, body string) (string, error) {
	var release *github.RepositoryRelease
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}

	data := &github.RepositoryRelease{
		Name:       github.String(title),
		TagName:    github.String(ctx.Git.CurrentTag),
		Body:       github.String(body),
		Draft:      github.Bool(ctx.Config.Release.Draft),
		Prerelease: github.Bool(ctx.PreRelease),
	}
	if ctx.Config.Release.DiscussionCategoryName != "" {
		data.DiscussionCategoryName = github.String(ctx.Config.Release.DiscussionCategoryName)
	}

	release, _, err = c.client.Repositories.GetReleaseByTag(
		ctx,
		ctx.Config.Release.GitHub.Owner,
		ctx.Config.Release.GitHub.Name,
		ctx.Git.CurrentTag,
	)
	if err != nil {
		release, _, err = c.client.Repositories.CreateRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			data,
		)
	} else {
		// keep the pre-existing release notes
		if release.GetBody() != "" {
			data.Body = release.Body
		}
		release, _, err = c.client.Repositories.EditRelease(
			ctx,
			ctx.Config.Release.GitHub.Owner,
			ctx.Config.Release.GitHub.Name,
			release.GetID(),
			data,
		)
	}
	log.WithField("url", release.GetHTMLURL()).Info("release updated")
	githubReleaseID := strconv.FormatInt(release.GetID(), 10)
	return githubReleaseID, err
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

func (c *githubClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
	file *os.File,
) error {
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
	if err == nil {
		return nil
	}
	if resp != nil && resp.StatusCode == 422 {
		return err
	}
	return RetriableError{err}
}

// getMilestoneByTitle returns a milestone by title.
func (c *githubClient) getMilestoneByTitle(ctx *context.Context, repo Repo, title string) (*github.Milestone, error) {
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
