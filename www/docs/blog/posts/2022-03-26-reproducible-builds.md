---
date: 2022-03-26
slug: reproducible-builds
categories:
  - tutorials
authors:
  - caarlos0
---

# Reproducible Builds

GoReleaser can help you, to some extent, to have reproducible builds.

![](https://carlosbecker.com/posts/goreleaser-reproducible-buids/c4824165-c6e2-40df-b4b5-8abe443195ce.png)

<!-- more -->

## **What are reproducible builds?**

According to [Reproducible-Builds.org](https://reproducible-builds.org/docs/definition/):

> A build is reproducible if given the same source code, build environment and build instructions, any party can recreate bit-by-bit identical copies of all specified artifacts.

So, things we need to pay attention here are:

- the source is the same
- the dependencies are the same, in the same versions
- `chtimes` are the same
- build path is the same
- any other tools needed to compile must be the same, in the same versions

While this might sound complicated, rest assured GoReleaser can help you with most of these items!

## **Reproducible Builds with GoReleaser**

GoReleaser will by default inject a `ldflag` with the current timestamp as `main.date`, which you can use to display build time information. We will want to change that to use some fixed date, for instance, the date of the commit being built.

To avoid changes from one machine to another, we will also want to use `-trimpath`.

Finally, we'll want to make sure the repo code haven't changed, e.g., when building a tag, we want to make sure it wasn't deleted and pushed again (i.e., moved).

We can achieve that with a config that looks like this:

```yaml
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    mod_timestamp: "{{ .CommitTimestamp }}"
    flags:
      - -trimpath
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{ .CommitDate }}

gomod:
  proxy: true
```

From now on, we basically only need to ensure the Go version is the same.

That's out of the scope of GoReleaser's scope, but easy enough to do in GitHub Actions by pinning to a specific version of Go.

So, there you have it: reproducible Go binary builds with GoReleaser!
