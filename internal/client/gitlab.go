package client

import (
	"cmp"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/changelog"
	"github.com/goreleaser/goreleaser/v2/internal/retryx"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

const DefaultGitLabDownloadURL = "https://gitlab.com"

var (
	_ Client            = &gitlabClient{}
	_ PullRequestOpener = &gitlabClient{}
	_ ReleaseChecker    = &gitlabClient{}
)

type gitlabClient struct {
	client   *gitlab.Client
	authType gitlab.AuthType

	isV17OrLater bool
}

// gitlabDo wraps a go-gitlab SDK call with retry logic.
func gitlabDo[T any](ctx *context.Context, fn func() (T, *gitlab.Response, error)) (T, *gitlab.Response, error) {
	var result T
	var resp *gitlab.Response
	err := retryx.Do(ctx, ctx.Config.Retry, func() error {
		var err error
		result, resp, err = fn()
		return gitlabError(err, resp)
	}, retryx.IsRetriable)
	return result, resp, err
}

// gitlabError wraps an error from the go-gitlab SDK into a retryx.HTTPError,
// translating the rate-limit headers of a 429 into a RetryAfter the retry layer
// can honor. The SDK's own retry layer is disabled (see newGitLab), so this is
// the only place that reads them.
func gitlabError(err error, resp *gitlab.Response) error {
	if err == nil {
		return nil
	}
	he := retryx.HTTPError{Err: err}
	if r := must(resp).Response; r != nil {
		he.Status = r.StatusCode
		if he.Status == http.StatusTooManyRequests {
			he.RetryAfter = rateLimitRetryAfter(r.Header)
		}
	}
	return he
}

// rateLimitRetryAfter returns how long to wait before retrying a rate-limited
// request, from either the RateLimit-Reset (unix timestamp) or the Retry-After
// (seconds) header. It returns 0 when neither header gives a usable value.
func rateLimitRetryAfter(header http.Header) time.Duration {
	if v, err := strconv.ParseInt(header.Get("RateLimit-Reset"), 10, 64); err == nil && v > 0 {
		if wait := time.Until(time.Unix(v, 0)); wait > 0 {
			return wait
		}
	}
	if v, err := strconv.Atoi(header.Get("Retry-After")); err == nil && v > 0 {
		return time.Duration(v) * time.Second
	}
	return 0
}

// newGitLab returns a gitlab client implementation.
func newGitLab(ctx *context.Context, token string, opts ...gitlab.ClientOptionFunc) (*gitlabClient, error) {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			//nolint:gosec
			InsecureSkipVerify: ctx.Config.GitLabURLs.SkipTLSVerify,
		},
	}
	options := append([]gitlab.ClientOptionFunc{
		gitlab.WithHTTPClient(&http.Client{
			Transport: transport,
		}),
		gitlab.WithRequestOptions(gitlab.WithContext(ctx)),
		// the SDK retries 429s and 5xx on its own, with its own budget and
		// backoff. retryx does that too, honoring the user configuration, so
		// let it own the retries.
		gitlab.WithoutRetries(),
	}, opts...)
	if ctx.Config.GitLabURLs.API != "" {
		apiURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.API)
		if err != nil {
			return nil, fmt.Errorf("templating GitLab API URL: %w", err)
		}

		options = append(options, gitlab.WithBaseURL(apiURL))
	}

	var client *gitlab.Client
	var authType gitlab.AuthType
	var err error
	if checkUseJobToken(*ctx, token) {
		client, err = gitlab.NewJobClient(token, options...)
		authType = gitlab.JobToken
	} else {
		client, err = gitlab.NewClient(token, options...)
		authType = gitlab.PrivateToken
	}
	if err != nil {
		return &gitlabClient{}, err
	}

	return &gitlabClient{
		client:       client,
		authType:     authType,
		isV17OrLater: isV17(ctx, client),
	}, nil
}

// versionRetry bounds the GitLab version probe. The probe is best effort: on
// failure goreleaser only assumes a GitLab older than v17. It must not spend
// the retry budget of the whole release, which defaults to 10 attempts with a
// 5m max delay, and would leave goreleaser looking wedged for ~25 minutes.
var versionRetry = config.Retry{
	Attempts: 3,
	Delay:    500 * time.Millisecond,
	MaxDelay: 2 * time.Second,
}

func isV17(ctx *context.Context, client *gitlab.Client) bool {
	v := os.Getenv("CI_SERVER_VERSION")
	if v == "" {
		var version *gitlab.Version
		if err := retryx.Do(ctx, versionRetry, func() error {
			var resp *gitlab.Response
			var err error
			version, resp, err = client.Version.GetVersion(nil)
			return gitlabError(err, resp)
		}, retryx.IsRetriable); err != nil {
			log.WithError(err).Warn("could not get gitlab version")
			return false
		}
		v = version.Version
	}
	vv, err := semver.NewVersion(v)
	if err != nil {
		log.WithError(err).Warn("could not parse gitlab version")
		return false
	}
	return vv.GreaterThanEqual(semver.New(17, 0, 0, "", ""))
}

func (c *gitlabClient) checkIsPrivateToken() error {
	if c.authType == gitlab.PrivateToken {
		return nil
	}
	return errors.New("the necessary APIs are not available when using CI_JOB_TOKEN")
}

func (c *gitlabClient) Changelog(ctx *context.Context, repo Repo, prev, current string) ([]ChangelogItem, error) {
	if err := c.checkIsPrivateToken(); err != nil {
		return nil, fmt.Errorf("changelog: %w", err)
	}
	cmpOpts := &gitlab.CompareOptions{
		From: &prev,
		To:   &current,
	}
	result, _, err := gitlabDo(ctx, func() (*gitlab.Compare, *gitlab.Response, error) {
		return c.client.Repositories.Compare(repo.String(), cmpOpts)
	})
	var log []ChangelogItem
	if err != nil {
		return nil, err
	}

	for _, commit := range result.Commits {
		log = append(log, fillDeprecated(ChangelogItem{
			SHA:     commit.ID,
			Message: strings.Split(commit.Message, "\n")[0],
			Authors: append(
				[]Author{{
					Name:  commit.AuthorName,
					Email: commit.AuthorEmail,
				}},
				changelog.ExtractCoAuthors(commit.Message)...,
			),
		}))
	}
	return log, nil
}

// getDefaultBranch get the default branch
func (c *gitlabClient) getDefaultBranch(ctx *context.Context, repo Repo) (string, error) {
	if branch := gitlabCIDefaultBranch(repo); branch != "" {
		return branch, nil
	}
	if err := c.checkIsPrivateToken(); err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}
	projectID := gitlabProjectID(repo)
	p, res, err := gitlabDo(ctx, func() (*gitlab.Project, *gitlab.Response, error) {
		return c.client.Projects.GetProject(projectID, nil)
	})
	if err != nil {
		log := log.WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).Warn("error checking for default branch")
		return "", err
	}
	return p.DefaultBranch, nil
}

func gitlabCIDefaultBranch(repo Repo) string {
	branch := os.Getenv("CI_DEFAULT_BRANCH")
	if branch == "" {
		return ""
	}

	projectID := gitlabProjectID(repo)
	if projectID == "" {
		return ""
	}
	if os.Getenv("CI_PROJECT_PATH") == projectID || os.Getenv("CI_PROJECT_ID") == projectID {
		return branch
	}
	return ""
}

func gitlabProjectID(repo Repo) string {
	projectID := repo.Name
	if repo.Owner != "" {
		projectID = repo.Owner + "/" + projectID
	}
	return projectID
}

// checkBranchExists checks if a branch exists
func (c *gitlabClient) checkBranchExists(ctx *context.Context, repo Repo, branch string) (bool, error) {
	projectID := gitlabProjectID(repo)

	_, res, err := gitlabDo(ctx, func() (*gitlab.Branch, *gitlab.Response, error) {
		return c.client.Branches.GetBranch(projectID, branch)
	})
	if err != nil && (res == nil || res.StatusCode != 404) {
		log.WithError(err).
			Error("error verify branch existence")
		return false, err
	}

	return res != nil && res.StatusCode != 404, nil
}

// CloseMilestone closes a given milestone.
func (c *gitlabClient) CloseMilestone(ctx *context.Context, repo Repo, title string) error {
	milestone, err := c.getMilestoneByTitle(ctx, repo, title)
	if err != nil {
		return err
	}

	if milestone == nil {
		return ErrNoMilestoneFound{Title: title}
	}

	closeStateEvent := "close"

	opts := &gitlab.UpdateMilestoneOptions{
		Description: &milestone.Description,
		DueDate:     milestone.DueDate,
		StartDate:   milestone.StartDate,
		StateEvent:  &closeStateEvent,
		Title:       &milestone.Title,
	}

	_, _, err = gitlabDo(ctx, func() (*gitlab.Milestone, *gitlab.Response, error) {
		return c.client.Milestones.UpdateMilestone(
			repo.String(),
			milestone.ID,
			opts,
		)
	})

	return err
}

// CreateFile gets a file in the repository at a given path
// and updates if it exists or creates it for later pipes in the pipeline.
func (c *gitlabClient) CreateFile(
	ctx *context.Context,
	commitAuthor config.CommitAuthor,
	repo Repo,
	content []byte, // the content of the formula.rb
	fileName, // the path to the formula.rb
	message string, // the commit msg
) error {
	if err := c.checkIsPrivateToken(); err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	projectID := gitlabProjectID(repo)

	log.
		WithField("projectID", projectID).
		Debug("project id")

	var branch, defaultBranch string
	var branchExists bool
	var err error
	// Use the branch if given one
	if repo.Branch != "" {
		branch = repo.Branch
		branchExists, err = c.checkBranchExists(ctx, repo, branch)
		if err != nil {
			return err
		}

		// Retrieving default branch because we need it for `start_branch`
		if !branchExists {
			defaultBranch, err = c.getDefaultBranch(ctx, repo)
			if err != nil {
				return err
			}
		}

		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			WithField("branchExists", branchExists).
			Debug("using given branch")
	} else {
		// Try to get the default branch from the Git provider
		branch, err = c.getDefaultBranch(ctx, repo)
		if err != nil {
			return err
		}

		defaultBranch = branch
		branchExists = true

		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			Debug("no branch given, using default branch")
	}

	// If the branch doesn't exist, we need to check the default branch
	// because that's what we use as `start_branch` later if the file needs
	// to be created.
	opts := &gitlab.GetFileOptions{Ref: &defaultBranch}
	if branchExists {
		opts.Ref = &branch
	}

	// Check if the file already exists
	var res *gitlab.Response
	_, res, err = gitlabDo(ctx, func() (*gitlab.File, *gitlab.Response, error) {
		return c.client.RepositoryFiles.GetFile(projectID, fileName, opts)
	})
	if err != nil && (res == nil || res.StatusCode != 404) {
		log := log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).
			Error("could not get file")
		return err
	}

	log.
		WithField("projectID", projectID).
		WithField("branch", branch).
		WithField("fileName", fileName).
		Info("pushing file")

	stringContents := string(content)

	if res.StatusCode == 404 {
		// Create a new file because it's not already there
		log.
			WithField("projectID", projectID).
			WithField("branch", branch).
			WithField("fileName", fileName).
			Debug("file doesn't exist, creating it")

		createOpts := &gitlab.CreateFileOptions{
			AuthorName:    &commitAuthor.Name,
			AuthorEmail:   &commitAuthor.Email,
			Content:       &stringContents,
			Branch:        &branch,
			CommitMessage: &message,
		}

		// Branch not found, thus Gitlab requires a "start branch" to create the file
		if !branchExists {
			createOpts.StartBranch = &defaultBranch
		}

		fileInfo, _, err := gitlabDo(ctx, func() (*gitlab.FileInfo, *gitlab.Response, error) {
			return c.client.RepositoryFiles.CreateFile(projectID, fileName, createOpts)
		})
		if err != nil {
			log := log.
				WithField("fileName", fileName).
				WithField("branch", branch).
				WithField("projectID", projectID)
			if res != nil {
				log = log.WithField("statusCode", res.StatusCode)
			}
			log.WithError(err).
				Error("could not create file")
			return err
		}

		log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID).
			WithField("filePath", fileInfo.FilePath).
			Debug("created file")
		return nil
	}

	// Update the existing file
	log.
		WithField("fileName", fileName).
		WithField("branch", branch).
		WithField("projectID", projectID).
		Debug("file exists, updating it")

	updateOpts := &gitlab.UpdateFileOptions{
		AuthorName:    &commitAuthor.Name,
		AuthorEmail:   &commitAuthor.Email,
		Content:       &stringContents,
		Branch:        &branch,
		CommitMessage: &message,
	}

	// Branch not found, thus Gitlab requires a "start branch" to update the file
	if !branchExists {
		updateOpts.StartBranch = &defaultBranch
	}

	updateFileInfo, res, err := gitlabDo(ctx, func() (*gitlab.FileInfo, *gitlab.Response, error) {
		return c.client.RepositoryFiles.UpdateFile(projectID, fileName, updateOpts)
	})
	if err != nil {
		log := log.
			WithField("fileName", fileName).
			WithField("branch", branch).
			WithField("projectID", projectID)
		if res != nil {
			log = log.WithField("statusCode", res.StatusCode)
		}
		log.WithError(err).
			Error("error updating file")
		return err
	}

	log := log.
		WithField("fileName", fileName).
		WithField("branch", branch).
		WithField("projectID", projectID).
		WithField("filePath", updateFileInfo.FilePath)
	if res != nil {
		log = log.WithField("statusCode", res.StatusCode)
	}
	log.Debug("updated file")
	return nil
}

// CreateRelease creates a new release or updates it by keeping
// the release notes if it exists.
func (c *gitlabClient) CreateRelease(ctx *context.Context, body string) (releaseID string, err error) {
	title, err := tmpl.New(ctx).Apply(ctx.Config.Release.NameTemplate)
	if err != nil {
		return "", err
	}
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return "", err
	}
	projectID := gitlabName
	if ctx.Config.Release.GitLab.Owner != "" {
		projectID = ctx.Config.Release.GitLab.Owner + "/" + projectID
	}
	log.
		WithField("owner", ctx.Config.Release.GitLab.Owner).
		WithField("name", gitlabName).
		WithField("projectID", projectID).
		Debug("projectID")

	name := title
	tagName := ctx.Git.CurrentTag
	release, resp, err := gitlabDo(ctx, func() (*gitlab.Release, *gitlab.Response, error) {
		return c.client.Releases.GetRelease(projectID, tagName)
	})
	if err != nil && (resp == nil || (resp.StatusCode != 403 && resp.StatusCode != 404)) {
		return "", err
	}

	if resp != nil && (resp.StatusCode == 403 || resp.StatusCode == 404) {
		log.WithError(err).Debug("get release")

		description := body
		ref := ctx.Git.Commit
		gitURL := ctx.Git.URL

		log.
			WithField("name", name).
			WithField("description", description).
			WithField("ref", ref).
			WithField("url", gitURL).
			Debug("creating release")
		release, _, err = gitlabDo(ctx, func() (*gitlab.Release, *gitlab.Response, error) {
			return c.client.Releases.CreateRelease(projectID, &gitlab.CreateReleaseOptions{
				Name:        &name,
				Description: &description,
				Ref:         &ref,
				TagName:     &tagName,
			})
		})
		if err != nil {
			log.WithError(err).Debug("error creating release")
			return "", err
		}
		log.WithField("name", release.Name).Info("release created")
	} else {
		desc := body
		if release != nil {
			desc = getReleaseNotes(release.Description, body, ctx.Config.Release.ReleaseNotesMode)
		}

		release, _, err = gitlabDo(ctx, func() (*gitlab.Release, *gitlab.Response, error) {
			return c.client.Releases.UpdateRelease(projectID, tagName, &gitlab.UpdateReleaseOptions{
				Name:        &name,
				Description: &desc,
			})
		})
		if err != nil {
			log.WithError(err).Debug("error updating release")
			return "", err
		}

		log.WithField("name", release.Name).Info("release updated")
	}

	return tagName, err // gitlab references a tag in a repo by its name
}

func (c *gitlabClient) PublishRelease(_ *context.Context, _ string /* releaseID */) (err error) {
	// GitLab doesn't support draft releases. So a created release is already published.
	return nil
}

func (c *gitlabClient) CanRelease(ctx *context.Context) error {
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return err
	}
	projectID := gitlabName
	if ctx.Config.Release.GitLab.Owner != "" {
		projectID = ctx.Config.Release.GitLab.Owner + "/" + projectID
	}
	p, _, err := gitlabDo(ctx, func() (*gitlab.Project, *gitlab.Response, error) {
		return c.client.Projects.GetProject(projectID, nil)
	})
	if err != nil {
		return fmt.Errorf("could not check release permissions: %w", err)
	}
	if p.Permissions == nil {
		// Permissions field is absent for certain auth types; skip the check.
		return nil
	}
	var maxLevel gitlab.AccessLevelValue
	if p.Permissions.ProjectAccess != nil && p.Permissions.ProjectAccess.AccessLevel > maxLevel {
		maxLevel = p.Permissions.ProjectAccess.AccessLevel
	}
	if p.Permissions.GroupAccess != nil && p.Permissions.GroupAccess.AccessLevel > maxLevel {
		maxLevel = p.Permissions.GroupAccess.AccessLevel
	}
	if maxLevel < gitlab.DeveloperPermissions {
		return fmt.Errorf("token does not have developer or higher permissions for %s", projectID)
	}
	return nil
}

func (c *gitlabClient) ReleaseURLTemplate(ctx *context.Context) (string, error) {
	var urlTemplate string
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return "", err
	}
	downloadURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.Download)
	if err != nil {
		return "", err
	}

	if ctx.Config.Release.GitLab.Owner != "" {
		urlTemplate = fmt.Sprintf(
			"%s/%s/%s/-/releases/{{ urlPathEscape .Tag }}/downloads/{{ .ArtifactName }}",
			downloadURL,
			ctx.Config.Release.GitLab.Owner,
			gitlabName,
		)
	} else {
		urlTemplate = fmt.Sprintf(
			"%s/%s/-/releases/{{ urlPathEscape .Tag }}/downloads/{{ .ArtifactName }}",
			downloadURL,
			gitlabName,
		)
	}
	return urlTemplate, nil
}

// Upload uploads a file into a release repository.
func (c *gitlabClient) Upload(
	ctx *context.Context,
	releaseID string,
	artifact *artifact.Artifact,
) error {
	// create new template and apply name field
	gitlabName, err := tmpl.New(ctx).Apply(ctx.Config.Release.GitLab.Name)
	if err != nil {
		return err
	}
	projectID := gitlabName
	// check if owner is empty
	if ctx.Config.Release.GitLab.Owner != "" {
		projectID = ctx.Config.Release.GitLab.Owner + "/" + projectID
	}

	return retryx.Do(ctx, ctx.Config.Retry, func() error {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		defer file.Close()

		var baseLinkURL string
		var linkURL string
		if ctx.Config.GitLabURLs.UsePackageRegistry || c.authType == gitlab.JobToken {
			log.WithField("file", file.Name()).Debug("uploading file as generic package")
			if _, resp, err := c.client.GenericPackages.PublishPackageFile(
				projectID,
				ctx.Config.ProjectName,
				ctx.Version,
				artifact.Name,
				file,
				nil,
			); err != nil {
				return gitlabError(err, resp)
			}

			baseLinkURL, err = c.client.GenericPackages.FormatPackageURL(
				projectID,
				ctx.Config.ProjectName,
				ctx.Version,
				artifact.Name,
			)
			if err != nil {
				return retryx.Unrecoverable(err)
			}
			linkURL = c.client.BaseURL().String() + baseLinkURL
		} else {
			log.WithField("file", file.Name()).Debug("uploading file as attachment")
			projectFile, resp, err := c.client.ProjectMarkdownUploads.UploadProjectMarkdown(
				projectID,
				file,
				filepath.Base(file.Name()),
				nil,
			)
			if err != nil {
				return gitlabError(err, resp)
			}

			baseLinkURL = projectFile.URL
			gitlabBaseURL, err := tmpl.New(ctx).Apply(ctx.Config.GitLabURLs.Download)
			if err != nil {
				return retryx.Unrecoverable(err)
			}

			linkURL = gitlabBaseURL + "/" + projectFile.FullPath
		}

		log.WithField("file", file.Name()).
			WithField("url", baseLinkURL).
			Debug("uploaded file")

		name := artifact.Name
		filename := "/" + name
		opt := &gitlab.CreateReleaseLinkOptions{
			Name: &name,
			URL:  &linkURL,
		}
		if c.isV17OrLater {
			opt.DirectAssetPath = &filename
		} else {
			opt.FilePath = &filename
		}

		releaseLink, resp, err := c.client.ReleaseLinks.CreateReleaseLink(
			projectID,
			releaseID,
			opt,
		)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusBadRequest {
				releaseLink, err = c.replaceReleaseLink(ctx, projectID, releaseID, name, opt, err)
				if err != nil {
					return err
				}
			} else {
				return gitlabError(err, resp)
			}
		}

		log.WithField("id", releaseLink.ID).
			WithField("url", releaseLink.DirectAssetURL).
			Debug("created release link")

		// for checksums.txt the field is nil, so we initialize it
		if artifact.Extra == nil {
			artifact.Extra = make(map[string]any)
		}

		return nil
	}, retryx.IsRetriable)
}

// replaceReleaseLink handles a failed release link creation that is likely
// caused by a link with the same name already existing: it deletes the existing
// link, if the user allowed it, and recreates it with the already uploaded URL.
//
// The failed creation does not return the ID of the existing link, so it has to
// be found in the release link list first.
func (c *gitlabClient) replaceReleaseLink(
	ctx *context.Context,
	projectID, releaseID, name string,
	opt *gitlab.CreateReleaseLinkOptions,
	createErr error,
) (*gitlab.ReleaseLink, error) {
	if !ctx.Config.Release.ReplaceExistingArtifacts {
		return nil, retryx.Unrecoverable(createErr)
	}

	link, err := c.getReleaseLinkByName(projectID, releaseID, name)
	if err != nil {
		return nil, errors.Join(createErr, err)
	}
	if link == nil {
		// the creation failed for some other reason.
		return nil, retryx.Unrecoverable(createErr)
	}

	if _, resp, err := c.client.ReleaseLinks.DeleteReleaseLink(
		projectID,
		releaseID,
		link.ID,
	); err != nil {
		return nil, errors.Join(createErr, gitlabError(err, resp))
	}

	log.WithField("id", link.ID).
		WithField("name", name).
		Debug("deleted existing release link")

	releaseLink, resp, err := c.client.ReleaseLinks.CreateReleaseLink(projectID, releaseID, opt)
	if err != nil {
		return nil, errors.Join(createErr, gitlabError(err, resp))
	}
	return releaseLink, nil
}

// getReleaseLinkByName returns the release link with the given name, or nil if
// the release has none.
func (c *gitlabClient) getReleaseLinkByName(
	projectID, releaseID, name string,
) (*gitlab.ReleaseLink, error) {
	opts := &gitlab.ListReleaseLinksOptions{}
	for {
		links, resp, err := c.client.ReleaseLinks.ListReleaseLinks(projectID, releaseID, opts)
		if err != nil {
			return nil, gitlabError(err, resp)
		}

		for _, link := range links {
			if link != nil && link.Name == name {
				return link, nil
			}
		}

		if resp == nil || resp.NextPage == 0 {
			return nil, nil
		}

		opts.Page = resp.NextPage
	}
}

// getMilestoneByTitle returns a milestone by title.
func (c *gitlabClient) getMilestoneByTitle(ctx *context.Context, repo Repo, title string) (*gitlab.Milestone, error) {
	opts := &gitlab.ListMilestonesOptions{
		Title: &title,
	}

	for {
		milestones, resp, err := gitlabDo(ctx, func() ([]*gitlab.Milestone, *gitlab.Response, error) {
			return c.client.Milestones.ListMilestones(repo.String(), opts)
		})
		if err != nil {
			return nil, err
		}

		for _, milestone := range milestones {
			if milestone != nil && milestone.Title == title {
				return milestone, nil
			}
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return nil, nil
}

// checkUseJobToken examines the context and given token, and determines if we
// should use NewJobClient vs NewClient
func checkUseJobToken(ctx context.Context, token string) bool {
	// The CI_JOB_TOKEN env var is set automatically in all GitLab runners.
	// If this comes back as empty, we aren't in a functional GitLab runner
	ciToken := os.Getenv("CI_JOB_TOKEN")
	if ciToken == "" {
		return false
	}

	// We only want to use the JobToken client if we have specified
	// UseJobToken. Older versions of GitLab don't work with this, so we
	// want to be specific
	if ctx.Config.GitLabURLs.UseJobToken {
		// We may be creating a new client with a non-CI_JOB_TOKEN, for
		// things like Homebrew publishing. We can't use the
		// CI_JOB_TOKEN there
		return token == ciToken
	}
	return false
}

func (c *gitlabClient) OpenPullRequest(
	ctx *context.Context,
	base, head Repo,
	title string,
	draft bool,
) (string, error) {
	if err := c.checkIsPrivateToken(); err != nil {
		return "", fmt.Errorf("open merge request: %w", err)
	}
	var targetProjectID int64
	if base.Owner != "" {
		fullProjectPath := fmt.Sprintf("%s/%s", base.Owner, base.Name)

		p, res, err := gitlabDo(ctx, func() (*gitlab.Project, *gitlab.Response, error) {
			return c.client.Projects.GetProject(fullProjectPath, nil)
		})
		if err != nil {
			log := log.WithField("project", fullProjectPath)
			if res != nil {
				log = log.WithField("statusCode", res.StatusCode)
			}
			log.WithError(err).Warn("error getting base project id")
			return "", err
		}
		targetProjectID = p.ID
	}

	base.Owner = cmp.Or(base.Owner, head.Owner)
	base.Name = cmp.Or(base.Name, head.Name)

	if base.Branch == "" {
		def, err := c.getDefaultBranch(ctx, base)
		if err != nil {
			return "", err
		}
		base.Branch = def
	}

	if draft {
		title = fmt.Sprintf("Draft: %s", title)
	}

	log.WithField("base", headString(base, Repo{})).
		WithField("head", headString(base, head)).
		WithField("draft", draft).
		Info("opening pull request")

	mrOptions := &gitlab.CreateMergeRequestOptions{
		SourceBranch: &head.Branch,
		TargetBranch: &base.Branch,
		Title:        &title,
		Description:  new(prFooter),
	}

	if targetProjectID != 0 {
		mrOptions.TargetProjectID = &targetProjectID
	}

	pr, _, err := gitlabDo(ctx, func() (*gitlab.MergeRequest, *gitlab.Response, error) {
		return c.client.MergeRequests.CreateMergeRequest(fmt.Sprintf("%s/%s", head.Owner, head.Name), mrOptions)
	})
	if err != nil {
		return "", fmt.Errorf("could not create pull request: %w", err)
	}
	log.WithField("url", pr.WebURL).Info("pull request created")
	return pr.WebURL, nil
}
