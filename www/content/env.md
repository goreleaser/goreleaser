---
title: Environment Variables
series: customization
hideFromIndex: true
weight: 19
---

Global environment variables to be passed down to all hooks and builds.

This is useful for `GO111MODULE`, for example. You can have your
`.goreleaser.yaml` file like the following:

```yaml
# .goreleaser.yml
env:
  - GO111MODULE=on
before:
  hooks:
    - go mod tidy
builds:
- binary: program
```

This way, both `go mod tidy` and the underlying `go build` will have
`GO111MODULE` set to `on`.

