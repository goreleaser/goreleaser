---
title: GitHub Actions
menu: true
weight: 141
---

GoReleaser can also be used within [GitHub Actions][actions].

For detailed intructions please follow GitHub Actions [workflow syntax][syntax].

You can create a workflow for pushing your releases by putting YAML
configuration to `.github/workflows/release.yml`.

Example workflow:
```yaml
on:
  push:
    tags:
      - 'v*'
name: GoReleaser
jobs:
  release:
    name: Release
    runs-on: ubuntu-latest
    #needs: [ test ]
    steps:
    - name: Check out code
      uses: actions/checkout@master
    - name: goreleaser
      uses: docker://goreleaser/goreleaser
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      with:
        args: release
      if: success()
```

This supports everything already supported by GoReleaser's [Docker image][docker].
Check the [install](/install) section for more details.

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
[syntax]: https://help.github.com/en/articles/workflow-syntax-for-github-actions#About-yaml-syntax-for-workflows
