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
    api: api.github.foo.bar
    upload: uploads.github.foo.bar
    download: github.foo.bar
```

If none are set, they default to GitHub's public URLs.

## The dist folder

By default, GoReleaser will create its artifacts in the `./dist` folder.
If you must, you can change it by setting it in the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
dist: another-folder-that-is-not-dist
```

## Using the `main.version`

GoReleaser always sets a `main.version` _ldflag_.
You can use it in your `main.go` file:

```go
package main

var version = "master"

func main() {
  println(version)
}
```

`version` will be set to the current Git tag (the `v` prefix is stripped) or the name of
the snapshot, if you're using the `--snapshot` flag.

You can override this by changing the `ldflags` option in the `build` section.

## Customizing Git

By default, GoReleaser uses full length commit hashes when setting a `main.commit`
_ldflag_ or creating filenames in `--snapshot` mode.

You can use short, 7 character long commit hashes by setting it in the `.goreleaser.yml`:

```yaml
# .goreleaser.yml
git:
  short_hash: true
```
