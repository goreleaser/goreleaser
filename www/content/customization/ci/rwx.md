---
title: "RWX"
weight: 95
---

Here is how to set up a release pipeline with [RWX](https://www.rwx.com).

The `preserve-git-dir` and `fetch-full-depth` options on the clone task are
required so GoReleaser can compute version metadata and changelogs.

```yaml {filename=".rwx/release.yml"}
on:
  github:
    push:
      if: ${{ event.git.tag != '' }}
      init:
        commit-sha: ${{ event.git.sha }}

base:
  image: ubuntu:24.04
  config: rwx/base 1.0.2

tasks:
  - key: code
    call: git/clone 2.0.7
    with:
      repository: https://github.com/YOUR-ORG/YOUR-REPO.git
      ref: ${{ init.commit-sha }}
      github-token: ${{ github.token }}
      preserve-git-dir: true
      fetch-full-depth: true

  - key: go
    call: golang/install 1.2.0
    with:
      go-version: "1.26"

  - key: goreleaser
    run: |
      arch="$(uname -m)"
      [ "$arch" = "aarch64" ] && arch="arm64"
      curl -fsSL "https://github.com/goreleaser/goreleaser/releases/download/${VERSION}/goreleaser_Linux_${arch}.tar.gz" \
        | sudo tar -xz -C /usr/local/bin goreleaser
    env:
      VERSION: v2.15.4

  - key: release
    use: [code, go, goreleaser]
    cache: false
    run: goreleaser release --clean
    env:
      GITHUB_TOKEN: ${{ github.token }}
```

To validate your configuration on pull requests, run a snapshot release that
skips publishing:

```yaml {filename=".rwx/pull-request.yml"}
on:
  github:
    pull_request:
      init:
        commit-sha: ${{ event.git.sha }}

tasks:
  - key: snapshot
    use: [code, go, goreleaser]
    run: goreleaser release --snapshot --clean --skip=publish
```

If you build Docker images, enable `docker: true` on the release task and add a
Docker login step before `goreleaser release`, sourcing credentials from
secrets with `cache-key: excluded`.

See the [RWX GoReleaser guide](https://www.rwx.com/docs/guides/goreleaser) for
more details.
