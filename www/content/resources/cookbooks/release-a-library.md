---
title: "Release a library"
weight: 100
---

You might want to generate a changelog and release notes for a Go library
without publishing any binaries. GoReleaser has you covered!

Add `skip: true` to the build configuration:

```yaml {filename=".goreleaser.yaml"}
builds:
- skip: true
```
