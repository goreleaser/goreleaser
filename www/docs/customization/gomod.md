---
title: Go Modules
---

GoReleaser has support for creating verifiable builds.
A [verifiable build][vgo] is one that records enough information to be precise about exactly how to repeat it.
All dependencies are loaded via `proxy.golang.org`, and verified against the checksum database `sum.golang.org`.
A GoReleaser-created verifiable build will include module information in the resulting binary, which can be printed using `go version -m mybinary`.

Configuration options available are described below.

```yaml
# goreleaser.yml

gomod:
  # Proxy a module from proxy.golang.org, making the builds verifiable.
  # This will only be effective if running against a tag. Snapshots will ignore this setting.
  # Notice: for this to work your `build.main` must be a package, not a `.go` file.
  #
  # Default is false.
  proxy: true

  # If proxy is true, use these environment variables when running `go mod` commands (namely, `go mod tidy`).
  # Defaults to `os.Environ()`.
  env:
    - GOPROXY=https://proxy.golang.org,direct
    - GOSUMDB=sum.golang.org
    - GOPRIVATE=example.com/blah

  # Which Go binary to use.
  # Defaults to `go`.
  gobinary: go1.15
```

!!! tip
    You can use `debug.ReadBuildInfo()` to get the version/checksum/dependencies of the module.

[vgo]: https://research.swtch.com/vgo-repro
