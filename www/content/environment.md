---
title: Environment
weight: 20
menu: true
---

## GitHub Token

GoReleaser requires a GitHub API token with the `repo` scope selected to
deploy the artifacts to GitHub.
You can create one [here](https://github.com/settings/tokens/new).

This token should be added to the environment variables as `GITHUB_TOKEN`.
Here is how to do it with Travis CI:
[Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).

Alternatively, you can provide the GitHub token in a file. GoReleaser will check `~/.config/goreleaser/github_token` by default, you can change that in
the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
env_files:
  github_token: ~/.path/to/my/token
```

## GitHub Enterprise

You can use GoReleaser with GitHub Enterprise by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
github_urls:
  api: https://git.company.com/api/v3/
  upload: https://git.company.com/api/uploads/
  download: https://git.company.com/
```

If none are set, they default to GitHub's public URLs.

**IMPORTANT**: be careful with the URLs, they may change from one installation
to another. If they are wrong, goreleaser will fail at some point, so, make
sure they're right before opening an issue. See for example [#472][472].

[472]: https://github.com/goreleaser/goreleaser/issues/472

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
)

func main() {
  fmt.Printf("%v, commit %v, built at %v", version, commit, date)
}
```

You can override this by changing the `ldflags` option in the `build` section.
