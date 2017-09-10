---
title: Introduction
---

GoReleaser is a release automation tool for Golang projects, the goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to [download and execute it](#integration-with-ci) in your build script.
You can [customize](#release-customization) your release process by createing a `.goreleaser.yml` file.

The idea started with a
[simple shell script](https://github.com/goreleaser/old-go-releaser),
but it quickly became more complex and I also wanted to publish binaries via
Homebrew taps, which would made the script even more hacky, so I let go of
that and rewrote the whole thing in Go.
