# retryx Everywhere Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Finish the WIP `internal/retryx` package and apply retry logic to every external service call in goreleaser.

**Architecture:** Add `IsRetriableHTTPError` and `Unrecoverable` to `retryx`, then create per-client generic helpers (`githubDo`, `gitlabDo`, `giteaDo`) that adapt each SDK's triple-return pattern. Wrap all API-calling methods. Add `retryx.Do` to announcement pipes and the HTTP upload utility. Remove the now-unnecessary `RetriableError` type.

**Tech Stack:** Go generics, `avast/retry-go/v4`, `google/go-github/v84`, `gitlab.com/gitlab-org/api/client-go`, `code.gitea.io/sdk/gitea`

---

## Task 1: Core retryx Additions

**Files:**
- Modify: `internal/retryx/retryx.go`
- Modify: `internal/retryx/retryx_test.go`

- [ ] **Step 1: Add `IsRetriableHTTPError` and `Unrecoverable` to `retryx.go`**

Add `"net/http"` to imports and append these functions after `IsNetworkError`:

```go
// IsRetriableHTTPError returns true if the status code or error indicates a
// transient HTTP failure worth retrying.
func IsRetriableHTTPError(statusCode int, err error) bool {
	if IsNetworkError(err) {
		return true
	}
	return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}

// Unrecoverable wraps an error so that the retry loop stops immediately.
func Unrecoverable(err error) error {
	return retry.Unrecoverable(err)
}
```

- [ ] **Step 2: Write tests for `IsRetriableHTTPError`**

Add to `retryx_test.go`:

```go
func TestIsRetriableHTTPError(t *testing.T) {
	t.Run("network error", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(0, errors.New("connection reset")))
	})
	t.Run("500", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(500, errors.New("internal server error")))
	})
	t.Run("502", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(502, errors.New("bad gateway")))
	})
	t.Run("503", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(503, errors.New("service unavailable")))
	})
	t.Run("429", func(t *testing.T) {
		require.True(t, IsRetriableHTTPError(429, errors.New("rate limited")))
	})
	t.Run("404 not retriable", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(404, errors.New("not found")))
	})
	t.Run("422 not retriable", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(422, errors.New("unprocessable")))
	})
	t.Run("200 no error", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(200, nil))
	})
	t.Run("0 no error", func(t *testing.T) {
		require.False(t, IsRetriableHTTPError(0, nil))
	})
}

func TestUnrecoverable(t *testing.T) {
	err := Unrecoverable(errors.New("permanent"))
	var calls atomic.Int32
	result := Do(fastRetry(5), func() error {
		calls.Add(1)
		return err
	}, nil)
	require.ErrorContains(t, result, "permanent")
	require.Equal(t, int32(1), calls.Load())
}
```

- [ ] **Step 3: Run retryx tests**

Run: `go test ./internal/retryx/... -v -count=1`
Expected: All tests pass including the new ones.

- [ ] **Step 4: Commit**

```bash
git add internal/retryx/
git commit -m "feat(retryx): add IsRetriableHTTPError and Unrecoverable"
```

---

## Task 2: GitHub Client — Add Helper and Wrap All Methods

**Files:**
- Modify: `internal/client/github.go`
- Delete: `internal/client/github_retry.go`

- [ ] **Step 1: Add `githubDo` helper function**

Add this helper after the `githubClient` struct definition (after line 38):

```go
// githubDo wraps a go-github SDK call with retry logic.
// It captures the response for status-code-based retry decisions.
func githubDo[T any](ctx *context.Context, fn func() (T, *github.Response, error)) (T, error) {
	var resp *github.Response
	return retryx.DoWithData(ctx.Config.Retry, func() (T, error) {
		result, r, err := fn()
		resp = r
		return result, err
	}, func(err error) bool {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return retryx.IsRetriableHTTPError(code, err)
	})
}
```

- [ ] **Step 2: Refactor `GenerateReleaseNotes` to use `githubDo`**

Replace lines 123–133:

```go
// Before:
notes, err := retryx.DoWithData(
    ctx.Config.Retry,
    func() (*github.RepositoryReleaseNotes, error) {
        notes, _, err := c.client.Repositories.GenerateReleaseNotes(ctx, repo.Owner, repo.Name, &github.GenerateNotesOptions{
            TagName:         current,
            PreviousTagName: &prev,
        })
        return notes, err
    },
    retryx.IsNetworkError,
)
```

With:

```go
// After:
notes, err := githubDo(ctx, func() (*github.RepositoryReleaseNotes, *github.Response, error) {
    return c.client.Repositories.GenerateReleaseNotes(ctx, repo.Owner, repo.Name, &github.GenerateNotesOptions{
        TagName:         current,
        PreviousTagName: &prev,
    })
})
```

- [ ] **Step 3: Refactor `Changelog` to use `githubDo`**

Replace lines 148–159. Note: this method needs `resp.NextPage`, so capture it inside the closure:

```go
// After:
result, err := githubDo(ctx, func() (*github.CommitsComparison, *github.Response, error) {
    result, resp, err := c.client.Repositories.CompareCommits(ctx, repo.Owner, repo.Name, prev, current, opts)
    if err == nil {
        nextPage = resp.NextPage
    }
    return result, resp, err
})
```

- [ ] **Step 4: Refactor `authorsLookup` to use `githubDo`**

Replace lines 206–209:

```go
// After:
res, err := githubDo(ctx, func() (*github.UsersSearchResult, *github.Response, error) {
    return c.client.Search.Users(ctx, author.Email, nil)
})
```

- [ ] **Step 5: Wrap `getDefaultBranch`**

Replace lines 222–231. Capture `res` for the status code log:

```go
var res *github.Response
p, err := githubDo(ctx, func() (*github.Repository, *github.Response, error) {
    p, r, err := c.client.Repositories.Get(ctx, repo.Owner, repo.Name)
    res = r
    return p, r, err
})
if err != nil {
    log := log.WithField("projectID", repo.String())
    if res != nil {
        log = log.WithField("statusCode", res.StatusCode)
    }
    log.WithError(err).Warn("error checking for default branch")
    return "", err
}
return p.GetDefaultBranch(), nil
```

- [ ] **Step 6: Wrap `CloseMilestone` (line 251), `getPRTemplate` (line 271)**

For `CloseMilestone`, wrap the `EditMilestone` call:

```go
_, err = githubDo(ctx, func() (*github.Milestone, *github.Response, error) {
    return c.client.Issues.EditMilestone(ctx, repo.Owner, repo.Name, *milestone.Number, milestone)
})
return err
```

For `getPRTemplate`, wrap `GetContents`:

```go
func (c *githubClient) getPRTemplate(ctx *context.Context, repo Repo) (string, error) {
    content, err := githubDo(ctx, func() (*github.RepositoryContent, *github.Response, error) {
        content, _, resp, err := c.client.Repositories.GetContents(
            ctx, repo.Owner, repo.Name,
            ".github/PULL_REQUEST_TEMPLATE.md",
            &github.RepositoryContentGetOptions{Ref: repo.Branch},
        )
        return content, resp, err
    })
    if err != nil {
        return "", err
    }
    return content.GetContent()
}
```

- [ ] **Step 7: Wrap `OpenPullRequest` (line 313) and `SyncFork` (line 345)**

For `OpenPullRequest`, wrap `PullRequests.Create` and capture `res` for 422 check:

```go
var res *github.Response
pr, err := githubDo(ctx, func() (*github.PullRequest, *github.Response, error) {
    pr, r, err := c.client.PullRequests.Create(ctx, base.Owner, base.Name, &github.NewPullRequest{...})
    res = r
    return pr, r, err
})
if err != nil {
    if res != nil && res.StatusCode == http.StatusUnprocessableEntity {
        log.WithError(err).Warn("pull request validation failed")
        return nil
    }
    return fmt.Errorf("could not create pull request: %w", err)
}
```

For `SyncFork`, wrap `MergeUpstream`:

```go
var resp *github.Response
res, err := githubDo(ctx, func() (*github.RepoMergeUpstreamResult, *github.Response, error) {
    res, r, err := c.client.Repositories.MergeUpstream(ctx, head.Owner, head.Name, &github.RepoMergeUpstreamRequest{Branch: &branch})
    resp = r
    return res, r, err
})
if err != nil {
    return fmt.Errorf("%w: %s", err, bodyOf(resp))
}
```

- [ ] **Step 8: Wrap `CreateFile` SDK calls (lines 408, 414, 419, 431, 447)**

Wrap each individual SDK call inside `CreateFile` with `githubDo`. There are 5 calls:
1. `c.client.Repositories.GetBranch` — capture `res` for 404 check
2. `c.client.Git.GetRef`
3. `c.client.Git.CreateRef`
4. `c.client.Repositories.GetContents` — capture `res` for 404 check
5. `c.client.Repositories.UpdateFile`

Each follows the same pattern. Example for `GetBranch`:

```go
var res *github.Response
_, err := githubDo(ctx, func() (*github.Branch, *github.Response, error) {
    b, r, err := c.client.Repositories.GetBranch(ctx, repo.Owner, repo.Name, branch, 100)
    res = r
    return b, r, err
})
if err != nil && (res == nil || res.StatusCode != http.StatusNotFound) {
    return fmt.Errorf("could not get branch %q: %w", branch, err)
}
```

- [ ] **Step 9: Wrap `createOrUpdateRelease` (line 538), `findRelease` (line 571), `updateRelease` (line 584)**

For `createOrUpdateRelease`, wrap `Repositories.CreateRelease`:

```go
var resp *github.Response
release, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
    rel, r, err := c.client.Repositories.CreateRelease(ctx, ctx.Config.Release.GitHub.Owner, ctx.Config.Release.GitHub.Name, data)
    resp = r
    return rel, r, err
})
```

For `findRelease`, wrap `GetReleaseByTag`:

```go
release, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
    return c.client.Repositories.GetReleaseByTag(ctx, ctx.Config.Release.GitHub.Owner, ctx.Config.Release.GitHub.Name, name)
})
```

For `updateRelease`, wrap `EditRelease`:

```go
var resp *github.Response
release, err := githubDo(ctx, func() (*github.RepositoryRelease, *github.Response, error) {
    rel, r, err := c.client.Repositories.EditRelease(ctx, ctx.Config.Release.GitHub.Owner, ctx.Config.Release.GitHub.Name, id, data)
    resp = r
    return rel, r, err
})
```

- [ ] **Step 10: Wrap `Upload` with `retryx.Do` (special case)**

Replace the entire Upload method body to wrap with `retryx.Do`. Use `nil` retryIf (retry all errors) and `retryx.Unrecoverable` for permanent failures:

```go
func (c *githubClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact) error {
	c.checkRateLimit(ctx, time.Sleep)
	githubReleaseID, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return err
	}

	return retryx.Do(ctx.Config.Retry, func() error {
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
			&github.UploadOptions{Name: artifact.Name},
			file,
		)
		if err == nil {
			return nil
		}

		githubErrLogger(resp, err).
			WithField("name", artifact.Name).
			WithField("release-id", releaseID).
			Warn("upload failed")

		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			if !ctx.Config.Release.ReplaceExistingArtifacts {
				return retryx.Unrecoverable(err)
			}
			if delErr := c.deleteReleaseArtifact(ctx, githubReleaseID, artifact.Name, 1); delErr != nil {
				return retryx.Unrecoverable(delErr)
			}
		}
		return err
	}, nil)
}
```

- [ ] **Step 11: Wrap `deleteReleaseArtifact` (lines 615, 635), `deleteExistingDraftRelease` (line 749), `findDraftRelease` (line 770), `getMilestoneByTitle` (line 716)**

For `deleteReleaseArtifact`, wrap `ListReleaseAssets` and `DeleteReleaseAsset`:

```go
assets, err := githubDo(ctx, func() ([]*github.ReleaseAsset, *github.Response, error) {
    return c.client.Repositories.ListReleaseAssets(ctx, ctx.Config.Release.GitHub.Owner, ctx.Config.Release.GitHub.Name, releaseID, &github.ListOptions{PerPage: 100, Page: page})
})
```

```go
_, err := githubDo(ctx, func() (*github.Response, *github.Response, error) {
    resp, err := c.client.Repositories.DeleteReleaseAsset(ctx, ctx.Config.Release.GitHub.Owner, ctx.Config.Release.GitHub.Name, asset.GetID())
    return resp, resp, err
})
```

Note: `DeleteReleaseAsset` returns `(*github.Response, error)` — adapt by returning `resp` twice or by using `retryx.Do` directly instead.

For `deleteExistingDraftRelease`, wrap `DeleteRelease` similarly.

For `findDraftRelease` and `getMilestoneByTitle`, wrap the paginated `ListReleases`/`ListMilestones` calls, capturing `resp.NextPage` in the closure.

- [ ] **Step 12: Delete `github_retry.go`**

```bash
rm internal/client/github_retry.go
```

- [ ] **Step 13: Run GitHub client tests**

Run: `go test ./internal/client/... -v -count=1 -run GitHub`
Expected: All existing tests pass.

- [ ] **Step 14: Commit**

```bash
git add internal/client/github.go internal/client/github_retry.go
git commit -m "feat(github): wrap all API calls with retryx"
```

---

## Task 3: GitLab Client — Add Helper and Wrap All Methods

**Files:**
- Modify: `internal/client/gitlab.go`

- [ ] **Step 1: Add imports and `gitlabDo` helper**

Add `"github.com/goreleaser/goreleaser/v2/internal/retryx"` to imports. Add helper:

```go
func gitlabDo[T any](ctx *context.Context, fn func() (T, *gitlab.Response, error)) (T, error) {
	var resp *gitlab.Response
	return retryx.DoWithData(ctx.Config.Retry, func() (T, error) {
		result, r, err := fn()
		resp = r
		return result, err
	}, func(err error) bool {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return retryx.IsRetriableHTTPError(code, err)
	})
}
```

- [ ] **Step 2: Wrap `Changelog`, `getDefaultBranch`, `checkBranchExists`**

Each follows the same pattern. `Changelog` needs `resp.NextPage` captured. `checkBranchExists` needs `res.StatusCode` for 404 check.

Example for `Changelog`:

```go
var resp *gitlab.Response
result, err := gitlabDo(ctx, func() (*gitlab.Compare, *gitlab.Response, error) {
    r, rr, err := c.client.Repositories.Compare(repo.Owner+"/"+repo.Name, opts)
    resp = rr
    return r, rr, err
})
// Use resp.NextPage as before
```

- [ ] **Step 3: Wrap `CloseMilestone`, `getMilestoneByTitle`**

Same pattern — wrap the SDK call with `gitlabDo`.

- [ ] **Step 4: Wrap `CreateFile` SDK calls**

`CreateFile` has multiple SDK calls: `GetFile`, `CreateFile`, `UpdateFile`, `GetBranch`. Wrap each individually, capturing response for status code checks (404, etc.).

- [ ] **Step 5: Wrap `CreateRelease` SDK calls**

Wrap `GetRelease`, `CreateRelease`, `UpdateRelease`. Capture response for 403/404 checks.

- [ ] **Step 6: Wrap `Upload` with `retryx.Do` (special case)**

Similar pattern to GitHub Upload — use `retryx.Do` with `nil` retryIf and `retryx.Unrecoverable` for permanent errors. Replace `RetriableError` returns:

```go
// Where it currently says:
return RetriableError{err}
// Replace with just:
return err
```

And for non-retriable cases:
```go
return retryx.Unrecoverable(err)
```

- [ ] **Step 7: Wrap `OpenPullRequest` SDK calls**

Wrap `GetProject` and `CreateMergeRequest`.

- [ ] **Step 8: Run GitLab client tests**

Run: `go test ./internal/client/... -v -count=1 -run GitLab`
Expected: All existing tests pass.

- [ ] **Step 9: Commit**

```bash
git add internal/client/gitlab.go
git commit -m "feat(gitlab): wrap all API calls with retryx"
```

---

## Task 4: Gitea Client — Add Helper and Wrap All Methods

**Files:**
- Modify: `internal/client/gitea.go`

- [ ] **Step 1: Add imports and `giteaDo` helper**

Add retryx import. Add helper — note the Gitea SDK returns `(*gitea.Response, error)` or `(T, *gitea.Response, error)`:

```go
func giteaDo[T any](ctx *context.Context, fn func() (T, *gitea.Response, error)) (T, error) {
	var resp *gitea.Response
	return retryx.DoWithData(ctx.Config.Retry, func() (T, error) {
		result, r, err := fn()
		resp = r
		return result, err
	}, func(err error) bool {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return retryx.IsRetriableHTTPError(code, err)
	})
}
```

- [ ] **Step 2: Wrap `Changelog`, `CloseMilestone`, `getDefaultBranch`**

For `CloseMilestone`, capture `resp` for 404 check:

```go
var resp *gitea.Response
_, err := giteaDo(ctx, func() (*gitea.Milestone, *gitea.Response, error) {
    m, r, err := c.client.EditMilestoneByName(repo.Owner, repo.Name, title, opts)
    resp = r
    return m, r, err
})
if resp != nil && resp.StatusCode == http.StatusNotFound {
    return ErrNoMilestoneFound{Title: title}
}
```

- [ ] **Step 3: Wrap `CreateFile`, `createRelease`, `getExistingRelease`, `updateRelease`**

Wrap each SDK call. `CreateFile` has `GetContents`, `CreateFile`, `UpdateFile`.

- [ ] **Step 4: Wrap `Upload` with `retryx.Do` (special case)**

Replace `RetriableError` return with plain error return (inside retryx.Do):

```go
func (c *giteaClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact) error {
	giteaReleaseID, err := strconv.ParseInt(releaseID, 10, 64)
	if err != nil {
		return err
	}

	return retryx.Do(ctx.Config.Retry, func() error {
		file, err := os.Open(artifact.Path)
		if err != nil {
			return retryx.Unrecoverable(err)
		}
		defer file.Close()

		_, _, err = c.client.CreateReleaseAttachment(
			ctx.Config.Release.Gitea.Owner,
			ctx.Config.Release.Gitea.Name,
			giteaReleaseID, file, artifact.Name,
		)
		return err
	}, nil)
}
```

- [ ] **Step 5: Run Gitea client tests**

Run: `go test ./internal/client/... -v -count=1 -run Gitea`
Expected: All existing tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/client/gitea.go
git commit -m "feat(gitea): wrap all API calls with retryx"
```

---

## Task 5: Remove `RetriableError` and Fix Mock/Tests

**Files:**
- Modify: `internal/client/client.go`
- Modify: `internal/client/mock.go`
- Modify: `internal/pipe/release/release_test.go` (if needed)

**Depends on:** Tasks 2, 3, 4 (all RetriableError usages in real clients removed first)

- [ ] **Step 1: Delete `RetriableError` type from `client.go`**

Remove lines 187–194:

```go
// RetriableError is an error that will cause the action to be retried.
type RetriableError struct {
	Err error
}

func (e RetriableError) Error() string {
	return e.Err.Error()
}
```

- [ ] **Step 2: Update `Mock.Upload` to not use `RetriableError`**

Replace the `FailFirstUpload` handling — since real clients now retry internally, the mock should simulate success after internal retry:

```go
if c.FailFirstUpload {
    c.FailFirstUpload = false
    // Real clients retry internally via retryx; mock simulates this.
}
```

- [ ] **Step 3: Run release pipe tests**

Run: `go test ./internal/pipe/release/... -v -count=1`
Expected: `TestRunPipeUploadRetry` passes — mock now simulates successful internal retry.

- [ ] **Step 4: Run full client tests**

Run: `go test ./internal/client/... -v -count=1`
Expected: All pass — no code references `RetriableError` anymore.

- [ ] **Step 5: Commit**

```bash
git add internal/client/client.go internal/client/mock.go
git commit -m "refactor: remove RetriableError, retry is now internal to clients"
```

---

## Task 6: Direct HTTP Announcement Pipes

**Files:**
- Modify: `internal/pipe/discord/discord.go`
- Modify: `internal/pipe/telegram/telegram.go`
- Modify: `internal/pipe/discourse/discourse.go`
- Modify: `internal/pipe/mattermost/mattermost.go`
- Modify: `internal/pipe/webhook/webhook.go`
- Modify: `internal/pipe/opencollective/opencollective.go`
- Modify: `internal/pipe/mcp/mcp.go`

All follow the same pattern. Add `retryx` import and wrap the HTTP call.

- [ ] **Step 1: Wrap Discord**

In `discord.go`, wrap lines 101–109 with `retryx.Do`:

```go
return retryx.Do(ctx.Config.Retry, func() error {
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != 204 && resp.StatusCode != 200 {
        return fmt.Errorf("%s", resp.Status)
    }
    return nil
}, func(err error) bool {
    return retryx.IsNetworkError(err)
})
```

Note: we use `IsNetworkError` (not `IsRetriableHTTPError`) because the non-200 status is already converted to an error by the closure, and we don't want to retry 4xx errors. For 5xx, the error message will contain the status, and we check network-level failures.

Actually, for better behavior, capture the status code:

```go
var statusCode int
return retryx.Do(ctx.Config.Retry, func() error {
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        statusCode = 0
        return err
    }
    defer resp.Body.Close()
    statusCode = resp.StatusCode

    if resp.StatusCode != 204 && resp.StatusCode != 200 {
        return fmt.Errorf("%s", resp.Status)
    }
    return nil
}, func(err error) bool {
    return retryx.IsRetriableHTTPError(statusCode, err)
})
```

- [ ] **Step 2: Wrap Telegram**

Same pattern as Discord, applied to `telegram.go`. The success check is `resp.StatusCode >= http.StatusBadRequest` — adapt accordingly.

- [ ] **Step 3: Wrap Discourse, Mattermost**

Same pattern. Both use `http.DefaultClient.Do(req)` with status checks.

- [ ] **Step 4: Wrap Webhook**

Similar pattern but uses a custom `client` (with TLS config), not `http.DefaultClient`. Wrap `client.Do(req)`.

- [ ] **Step 5: Wrap OpenCollective**

Wrap `http.DefaultClient.Do(req)` in the `doMutation()` method.

- [ ] **Step 6: Wrap MCP**

Wrap `client.Do(req)` in the `Publish()` method.

- [ ] **Step 7: Run announcement pipe tests**

Run: `go test ./internal/pipe/discord/... ./internal/pipe/telegram/... ./internal/pipe/discourse/... ./internal/pipe/mattermost/... ./internal/pipe/webhook/... ./internal/pipe/opencollective/... ./internal/pipe/mcp/... -v -count=1`
Expected: All pass.

- [ ] **Step 8: Commit**

```bash
git add internal/pipe/discord/ internal/pipe/telegram/ internal/pipe/discourse/ internal/pipe/mattermost/ internal/pipe/webhook/ internal/pipe/opencollective/ internal/pipe/mcp/
git commit -m "feat(announce): add retryx to direct HTTP announcement pipes"
```

---

## Task 7: SDK-Wrapped Announcement Pipes + LinkedIn

**Files:**
- Modify: `internal/pipe/slack/slack.go`
- Modify: `internal/pipe/mastodon/mastodon.go`
- Modify: `internal/pipe/teams/teams.go`
- Modify: `internal/pipe/reddit/reddit.go`
- Modify: `internal/pipe/twitter/twitter.go`
- Modify: `internal/pipe/bluesky/bluesky.go`
- Modify: `internal/pipe/linkedin/linkedin.go`

These use third-party SDKs — we can only retry on error, not response code.

- [ ] **Step 1: Wrap Slack**

In `slack.go`, wrap the `PostWebhook` call at line 74:

```go
return retryx.Do(ctx.Config.Retry, func() error {
    return slack.PostWebhook(cfg.Webhook, wm)
}, retryx.IsNetworkError)
```

- [ ] **Step 2: Wrap Mastodon**

Wrap the `client.PostStatus` call:

```go
_, err = retryx.DoWithData(ctx.Config.Retry, func() (*mastodon.Status, error) {
    return client.PostStatus(ctx, &mastodon.Toot{Status: msg})
}, retryx.IsNetworkError)
```

- [ ] **Step 3: Wrap Teams, Reddit, Twitter**

Same pattern — wrap the SDK call with `retryx.Do`, using `retryx.IsNetworkError`.

- [ ] **Step 4: Wrap Bluesky**

Wrap both `ServerCreateSession` and `RepoCreateRecord` calls.

- [ ] **Step 5: Wrap LinkedIn**

In `linkedin.go`, wrap the `c.Share(ctx, message)` call in `Announce`:

```go
url, err := retryx.DoWithData(ctx.Config.Retry, func() (string, error) {
    return c.Share(ctx, message)
}, retryx.IsNetworkError)
```

- [ ] **Step 6: Run SDK pipe tests**

Run: `go test ./internal/pipe/slack/... ./internal/pipe/mastodon/... ./internal/pipe/teams/... ./internal/pipe/reddit/... ./internal/pipe/twitter/... ./internal/pipe/bluesky/... ./internal/pipe/linkedin/... -v -count=1`
Expected: All pass.

- [ ] **Step 7: Commit**

```bash
git add internal/pipe/slack/ internal/pipe/mastodon/ internal/pipe/teams/ internal/pipe/reddit/ internal/pipe/twitter/ internal/pipe/bluesky/ internal/pipe/linkedin/
git commit -m "feat(announce): add retryx to SDK-wrapped announcement pipes"
```

---

## Task 8: HTTP Upload Utility

**Files:**
- Modify: `internal/http/http.go`

- [ ] **Step 1: Add retryx import and wrap `executeHTTPRequest`**

Wrap the `client.Do(req)` call inside `executeHTTPRequest` with retry logic. Since this function returns `(*http.Response, error)`, wrap the entire body:

```go
func executeHTTPRequest(ctx *context.Context, upload *config.Upload, req *h.Request, check ResponseChecker) (*h.Response, error) {
	client, err := getHTTPClient(upload)
	if err != nil {
		return nil, err
	}

	var resp *h.Response
	err = retryx.Do(ctx.Config.Retry, func() error {
		log.Debugf("executing request: %s %s (headers: %v)", req.Method, req.URL, req.Header)
		var reqErr error
		resp, reqErr = client.Do(req)
		if reqErr != nil {
			select {
			case <-ctx.Done():
				return retryx.Unrecoverable(ctx.Err())
			default:
			}
			return reqErr
		}
		if err := check(resp); err != nil {
			resp.Body.Close()
			return err
		}
		return nil
	}, func(err error) bool {
		code := 0
		if resp != nil {
			code = resp.StatusCode
		}
		return retryx.IsRetriableHTTPError(code, err)
	})

	return resp, err
}
```

Note: caller still gets `resp` for body reading. `check` function validates the response — if it returns an error for 4xx, `IsRetriableHTTPError` won't retry it.

- [ ] **Step 2: Run HTTP upload tests**

Run: `go test ./internal/http/... -v -count=1`
Expected: All pass.

- [ ] **Step 3: Commit**

```bash
git add internal/http/http.go
git commit -m "feat(http): add retryx to HTTP upload utility"
```

---

## Task 9: Lint, Full Test Suite, Documentation

**Files:**
- Modify: `www/content/customization/general/retry.md`

**Depends on:** All previous tasks.

- [ ] **Step 1: Run linter**

Run: `golangci-lint run --tests --fix ./...`
Expected: No errors (auto-fixes applied if needed).

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass, including `TestRunPipeUploadRetry`.

- [ ] **Step 3: Update retry documentation**

Update `www/content/customization/general/retry.md` to mention that retry now applies to all external services: git providers (GitHub, GitLab, Gitea), announcement pipes, and HTTP uploads.

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "docs: update retry documentation for all services"
```
