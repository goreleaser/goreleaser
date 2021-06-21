---
title: Environment
---

## API Tokens

GoReleaser requires either a GitHub API token with the `repo` scope selected to
deploy the artifacts to GitHub **or** a GitLab API token with `api` scope **or** a Gitea API token.
You can create one [here](https://github.com/settings/tokens/new) for GitHub
or [here](https://gitlab.com/profile/personal_access_tokens) for GitLab
or in `Settings | Applications | Generate New Token` page of your Gitea instance.

This token should be added to the environment variables as `GITHUB_TOKEN` or `GITLAB_TOKEN` or `GITEA_TOKEN` respecively.
Here is how to do it with Travis CI:
[Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).

Alternatively, you can provide the GitHub/GitLab token in a file.
GoReleaser will check `~/.config/goreleaser/github_token`, `~/.config/goreleaser/gitlab_token`
and `~/.config/goreleaser/gitea_token` by default, you can change that in
the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
env_files:
  # use only one or release will fail!
  github_token: ~/.path/to/my/gh_token
  gitlab_token: ~/.path/to/my/gl_token
  gitea_token: ~/.path/to/my/gitea_token
```

!!! info
    you can define multiple env files, but the release process will fail
    because multiple tokens are defined. Use only one.

## GitHub Enterprise

You can use GoReleaser with GitHub Enterprise by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
github_urls:
  api: https://git.company.com/api/v3/
  upload: https://git.company.com/api/uploads/
  download: https://git.company.com/
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

If none are set, they default to GitHub's public URLs.

## GitLab Enterprise or private hosted

You can use GoReleaser with GitLab Enterprise by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
gitlab_urls:
  api: https://gitlab.mycompany.com/api/v4/
  download: https://gitlab.company.com
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

If none are set, they default to GitLab's public URLs.

## Gitea

You can use GoReleaser with Gitea by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
gitea_urls:
  api: https://gitea.myinstance.com/api/v1/
  download: https://gitea.myinstance.com
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

## The dist folder

By default, GoReleaser will create its artifacts in the `./dist` folder.
If you must, you can change it by setting it in the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
dist: another-folder-that-is-not-dist
```

## Using the `main.version`

Default wise GoReleaser sets three _ldflags_:

- `main.version`: Current Git tag (the `v` prefix is stripped) or the name of
  the snapshot, if you're using the `--snapshot` flag
- `main.commit`: Current git commit SHA
- `main.date`: Date according [RFC3339](https://golang.org/pkg/time/#pkg-constants)

You can use it in your `main.go` file:

```go
package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
    builtBy = "unknown"
)

func main() {
  fmt.Printf("my app %s, commit %s, built at %s by %s", version, commit, date, builtBy)
}
```

You can override this by changing the `ldflags` option in the `build` section.

## Overriding Git Tags

You can force the [build tag](/customization/build/#define-build-tag)
and [previous changelog tag](/customization/release/#define-previous-tag)
using environment variables. This is useful in cases where one git commit
is referenced by multiple git tags.
