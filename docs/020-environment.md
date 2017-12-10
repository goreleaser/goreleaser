---
title: Environment
---

## GitHub Token

GoReleaser requires a GitHub API token with the `repo` scope selected to
deploy the artifacts to GitHub.
You can create one [here](https://github.com/settings/tokens/new).

This token should be added to the environment variables as `GITHUB_TOKEN`.
Here is how to do it with Travis CI:
[Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).

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
