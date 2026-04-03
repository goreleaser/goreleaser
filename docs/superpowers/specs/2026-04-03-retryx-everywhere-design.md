# Design: Finish retryx and Apply Everywhere

## Problem

The `internal/retryx` package provides retry-with-backoff (`Do`,
`DoWithData[T]`) but is only used in a handful of places. Most external service
calls — git clients (GitHub, GitLab, Gitea), announcement pipes (Discord,
Telegram, etc.), and upload utilities — have no retry logic. The GitHub client
is partially converted (3 methods) but most of its methods still lack retries.

Additionally, `RetriableError` (returned by Upload methods) is never checked by
callers — the release pipe ignores it entirely, causing `TestRunPipeUploadRetry`
to fail.

## Approach

1. Add a shared `IsRetriableHTTPError(statusCode, err)` predicate to `retryx`
2. Create per-client generic helpers that adapt triple-return SDK patterns to `retryx.DoWithData`
3. Wrap all external-service-calling methods with retryx
4. Fix the broken Upload retry flow
5. Use the global `ctx.Config.Retry` config everywhere (no per-pipe configs)

## Design

### 1. Core Retry Infrastructure (`internal/retryx`)

**New function — `IsRetriableHTTPError`:**

```go
// In internal/retryx/retryx.go
func IsRetriableHTTPError(statusCode int, err error) bool {
    if IsNetworkError(err) {
        return true
    }
    return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}
```

Retry triggers:

- Network errors (connection reset, timeout, refused, etc.)
- 5xx server errors (500, 502, 503, 504, etc.)
- 429 Too Many Requests (rate limiting)

**No changes** to `Do()` or `DoWithData[T]()`.

**Circular import resolution:** `retryx` handles network + 5xx + 429 checks only. No dependency on `client` package.

Each client's retry predicate for standard API calls:

```go
func isRetriable(statusCode int, err error) bool {
    return retryx.IsRetriableHTTPError(statusCode, err)
}
```

**Upload methods** use a different pattern — retry on ALL errors, with `retry.Unrecoverable()` for permanent failures:

```go
func (c *githubClient) Upload(ctx *context.Context, releaseID string, artifact *artifact.Artifact) error {
    return retryx.Do(ctx.Config.Retry, func() error {
        _, resp, err := c.client.Repositories.UploadReleaseAsset(...)
        if err == nil {
            return nil
        }
        if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
            if !ctx.Config.Release.ReplaceExistingArtifacts {
                return retryx.Unrecoverable(err) // permanent failure
            }
            c.deleteReleaseArtifact(...) // delete, next attempt will succeed
        }
        return err // transient — retry
    }, nil) // nil retryIf = retry all errors
}
```

### 2. Remove `RetriableError` (`internal/client/client.go`)

Delete the `RetriableError` type entirely. It was a signal for callers to retry, but now retry lives inside each client method via retryx. No caller needs this signal anymore.

For Upload methods specifically, use `retryx.Unrecoverable(err)` (a thin wrapper around `retry.Unrecoverable` from `avast/retry-go/v4`) to bail out of the retry loop for permanent failures (e.g., asset exists + replace disabled). All other errors are retried automatically.

Add to `internal/retryx/retryx.go`:

```go
func Unrecoverable(err error) error {
    return retry.Unrecoverable(err)
}
```

### 3. Per-Client Generic Helpers

Each git client gets a closure-capturing helper that adapts its SDK's triple-return `(T, *Response, error)` to retryx's `(T, error)`:

**GitHub** (`internal/client/github.go`):

```go
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

**GitLab** (`internal/client/gitlab.go`): Same pattern, `*gitlab.Response`.

**Gitea** (`internal/client/gitea.go`): Same pattern, `*gitea.Response`.

For void methods (Upload, CloseMilestone, etc.), use `retryx.Do()` with the same status-code-capturing closure pattern.

### 4. Methods to Wrap

**GitHub client** — all API-calling methods:

- `GenerateReleaseNotes` (already wrapped)
- `Changelog` (already wrapped — may need predicate update)
- `authorsLookup` (already wrapped — may need predicate update)
- `CreateRelease` — needs wrapping
- `PublishRelease` — needs wrapping
- `Upload` — needs wrapping (fixes broken RetriableError flow)
- `OpenPullRequest` — needs wrapping
- `SyncFork` — needs wrapping
- `CreateFile` / `CreateFiles` — needs wrapping
- `CloseMilestone` — needs wrapping
- `getDefaultBranch` — needs wrapping
- `deleteReleaseArtifact` — needs wrapping

**GitLab client** — all API-calling methods:

- `Changelog`, `getDefaultBranch`, `checkBranchExists`, `CloseMilestone`
- `CreateFile`, `CreateRelease`, `PublishRelease`, `Upload`
- `getMilestoneByTitle`, `OpenPullRequest`

**Gitea client** — all API-calling methods:

- `Changelog`, `CloseMilestone`, `getDefaultBranch`
- `CreateFile`, `createRelease`, `getExistingRelease`, `updateRelease`
- `CreateRelease`, `Upload`

### 5. Announcement Pipes

**Direct HTTP pipes** — wrap `http.Client.Do()` with `retryx.Do()`:

- `internal/pipe/discord/discord.go`
- `internal/pipe/telegram/telegram.go`
- `internal/pipe/discourse/discourse.go`
- `internal/pipe/mattermost/mattermost.go`
- `internal/pipe/webhook/webhook.go`
- `internal/pipe/linkedin/client.go`
- `internal/pipe/opencollective/opencollective.go`
- `internal/pipe/mcp/mcp.go`

Pattern: capture `resp.StatusCode` in closure, use `IsRetriableHTTPError` predicate.

**SDK-wrapped pipes** — wrap SDK call with `retryx.Do()`:

- `internal/pipe/slack/slack.go`
- `internal/pipe/mastodon/mastodon.go`
- `internal/pipe/teams/teams.go`
- `internal/pipe/reddit/reddit.go`
- `internal/pipe/twitter/twitter.go`
- `internal/pipe/bluesky/bluesky.go`

These can only retry on error (no response code access), so use `retryx.IsNetworkError` as predicate.

### 6. HTTP Upload Utility

**`internal/http/http.go`** — wrap `executeHTTPRequest()` / `client.Do(req)` with `retryx.Do()`. Retry on 5xx/429 + network errors. Non-retriable 4xx errors fail immediately.

### 7. Testing

- **Fix** `TestRunPipeUploadRetry` — now passes because Upload internally retries via retryx
- **Add** unit tests for `IsRetriableHTTPError`: network errors, 5xx, 429, non-retriable errors (4xx, nil)
- **Update** mock client: remove `RetriableError` usage from `FailFirstUpload` — instead have the mock return a plain error on first call, succeed on second (retryx in the caller handles the retry)
- **Verify** existing tests for docker/snapcraft/gomod/git still pass

### 8. Cleanup

- Delete `internal/client/github_retry.go` (empty file)
- **Delete `RetriableError` type** from `internal/client/client.go`
- Remove all `RetriableError` returns from GitHub, GitLab, Gitea Upload methods
- Update existing retryx call sites (GitHub's 3 methods) to use the new `githubDo` helper and updated predicate
- Update `www/content/customization/general/retry.md` to mention all retriable services

## Scope Exclusions

- S3/blob storage — AWS SDK has built-in retry
- Docker manifest/push, snapcraft, gomod, git clone — already use retryx (may update predicates for consistency but not required)
- Per-pipe retry configuration — everything uses global `ctx.Config.Retry`
