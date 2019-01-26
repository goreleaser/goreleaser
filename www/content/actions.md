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
    # at least GITHUB_TOKEN is required, you may need more though
    "DOCKER_USERNAME",
    "DOCKER_PASSWORD",
  ]
  args = "release"
  needs = ["is-tag"]
}
```

This should support *almost* everything already supported by GoReleaser's
[Docker image][docker]. Check the [install](/install) section for more details.

## What doesn't work

Projects that depend on `$GOPATH`. GitHub Actions override the `WORKDIR`
instruction and it seems like we can't override it.

In the future releases we may hack something together to work around this,
but, for now, only projects using Go modules are supported.

[actions]: https://github.com/features/actions
[docker]: https://hub.docker.com/r/goreleaser/goreleaser
