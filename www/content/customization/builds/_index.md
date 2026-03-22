---
title: "Build"
weight: 20
---

GoReleaser can build binaries and libraries for multiple languages and runtimes.
Each language is supported through a _builder_ interface: it receives a build
configuration and emits artifacts into the `dist` directory.

You can configure cross-compilation targets, build hooks, environment variables,
binary compression with UPX, and more.
