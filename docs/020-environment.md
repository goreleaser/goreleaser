---
title: Environment Setup
---

### GitHub Token

GoReleaser requires a GitHub API token with the `repo` scope checked to
deploy the artefacts to GitHub. You can create one
[here](https://github.com/settings/tokens/new).

This token should be added to the environment variables as `GITHUB_TOKEN`.
Here is how to do it with Travis CI:
[Defining Variables in Repository Settings](https://docs.travis-ci.com/user/environment-variables/#Defining-Variables-in-Repository-Settings).

### A note about `main.version`

GoReleaser always sets a `main.version` ldflag. You can use it in your
`main.go` file:

```go
package main

var version = "master"

func main() {
  println(version)
}
```

`version` will be the current Git tag (with `v` prefix stripped) or the name of
the snapshot if you're using the `--snapshot` flag.

You can override this by changing the `ldflags` option in the `build` section.
