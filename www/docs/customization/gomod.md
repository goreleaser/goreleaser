---
title: Go Modules
---

GoReleaser have some integrations with Go Modules, namely the proxy feature, which allow you to make your builds verifiable
via `go version -m mybinary`.

```yaml
# goreleaser.yml

# Proxy a module from proxy.golang.org, making the builds verifiable.
# This will only be effective if running against a tag. Snapshots will ignore this setting.
#
# Default is false.
gomod:
  proxy: true
```

!!! tip
    You can use `debug.ReadBuildInfo()` to get the version/checksum/dependencies of the module.
