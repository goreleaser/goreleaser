---
title: Global Hooks
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
```

If any of the hooks fails the build process is aborted.

