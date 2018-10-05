---
title: Global Hooks
series: customization
hideFromIndex: true
weight: 20
---

Some builds may need pre-build steps before building, e.g. `go generate`.
The `before` section allows for global hooks which will be executed before
the build is started.

The configuration is very simple, here is a complete example:

```yml
# .goreleaser.yml
before:
  hooks:
  - make clean
  - go generate ./...
  - go mod download
```

If any of the hooks fails the build process is aborted.

It is important to note that you can't have "complex" commands, like
`bash -c "echo foo bar"` or `foo | bar` or anything like that. If you need
to do things that are more complex than just calling a command with some
attributes, wrap it in a shell script or into your `Makefile`.
