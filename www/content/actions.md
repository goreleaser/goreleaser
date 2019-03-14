---
title: GitHub Actions
menu: true
weight: 141
---

GoReleaser can also be used within [GitHub Actions][actions].

You can create a workflow like this to push your releases.

```t
workflow "Release" {
  on = "push"
  resolves = ["goreleaser"]
}

action "is-tag" {
  uses = "actions/bin/filter@master"
  args = "tag"
}

action "goreleaser" {
  uses = "docker://goreleaser/goreleaser"
  secrets = [
    "GITHUB_TOKEN",
    "GORELEASER_GITHUB_TOKEN",
    # either GITHUB_TOKEN or GORELEASER_GITHUB_TOKEN is required
    "DOCKER_USERNAME",
    "DOCKER_PASSWORD",
  ]
  args = "release"
  needs = ["is-tag"]
}
```

This should support *almost* everything already supported by GoReleaser's
[Docker image][docker]. Check the [install](/install) section for more details.

If you need to push the homebrew tap to another repository, you'll need a
custom github token, for that, add a `GORELEASER_GITHUB_TOKEN` secret and
remove the default `GITHUB_TOKEN`. The default, auto-generated token only
has access to current the repo.

## What doesn't work

Projects that depend on `$GOPATH`. GitHub Actions override the `WORKDIR`
instruction and it seems like we can't override it.

In the future releases we may hack something together to work around this,
but, for now, only projects using Go modules are supported.

[actions]: https://github.com/features/actions
[docker]: https://hub.docker.com/r/goreleaser/goreleaser
