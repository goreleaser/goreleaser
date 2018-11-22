---
title: Introduction
weight: 1
menu: true
---

[GoReleaser](https://github.com/goreleaser/goreleaser) is a release automation
tool for Go projects, the goal is to simplify the build, release and
publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to
[download and execute it](/ci) in your build script. Of course, you can
also [install it locally](/install).

You can [customize](/customization) your release process by
creating a `.goreleaser.yml` file.

The idea started with a
[simple shell script](https://github.com/goreleaser/old-go-releaser),
but it quickly became more complex and I also wanted to publish binaries via
Homebrew taps, which would have made the script even more "hacky", so I let
go of that and rewrote the whole thing in Go.

