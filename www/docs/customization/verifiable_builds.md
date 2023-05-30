# Verifiable Builds

GoReleaser has support for creating verifiable builds. A [verifiable build][vgo]
is one that records enough information to be precise about exactly how to repeat
it. All dependencies are loaded via `proxy.golang.org`, and verified against the
checksum database `sum.golang.org`. A GoReleaser-created verifiable build will
include module information in the resulting binary, which can be printed using
`go version -m mybinary`.

Configuration options available are described below.

```yaml
# goreleaser.yaml

gomod:
  # Proxy a module from proxy.golang.org, making the builds verifiable.
  # This will only be effective if running against a tag. Snapshots will ignore
  # this setting.
  # Notice: for this to work your `build.main` must be a package, not a `.go` file.
  proxy: true

  # If proxy is true, use these environment variables when running `go mod`
  # commands (namely, `go mod tidy`).
  #
  # Default: `os.Environ()` merged with what you set the root `env` section.
  env:
    - GOPROXY=https://proxy.golang.org,direct
    - GOSUMDB=sum.golang.org
    - GOPRIVATE=example.com/blah

  # Sets the `-mod` flag value.
  #
  # Since: v1.7
  mod: mod

  # Which Go binary to use.
  #
  # Default: `go`.
  gobinary: go1.17
```

!!! tip

    You can use `debug.ReadBuildInfo()` to get the version/checksum/dependencies
    of the module.

!!! warning

    VCS Info will not be embedded in the binary, as in practice it is not being
    built from the source, but from the Go Mod Proxy.

[vgo]: https://research.swtch.com/vgo-repro
