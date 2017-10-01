---
title: Builds
---

Builds can be customized in multiple ways. You can specify for which `GOOS` and
`GOARCH` binaries are generated, and you can changed the name of the binary, flags, `ldflags`, hooks, etc.

Here is a commented `builds` section with all fields specified:

```yml
# .goreleaser.yml
builds:
  # You can have multiple builds defined as a yaml list
  -
    # Path to main.go file or main package.
    # Default is `.`.
    main: ./cmd/main.go

    # Name of the binary.
    # Default is the name of the project directory.
    binary: program

    # Set flags for custom build tags.
    # Default is empty.
    flags: -tags dev

    # Custom ldflags template.
    # This is parsed with the Go template engine and the following variables
    # are available:
    # - Date
    # - Commit
    # - Tag
    # - Version (Git tag without `v` prefix)
    # Date format is `2006-01-02_15:04:05`.
    # Default is `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`.
    ldflags: -s -w -X main.build={{.Version}}

    # Custom environment variables to be set during the builds.
    # Default is empty.
    env:
      - CGO_ENABLED=0

    # GOOS list to build for.
    # For more info refer to: https://golang.org/doc/install/source#environment
    # Defaults are darwin and linux.
    goos:
      - freebsd
      - windows

    # GOARCH to build for.
    # For more info refer to: https://golang.org/doc/install/source#environment
    # Defaults are 386 and amd64.
    goarch:
      - amd64
      - arm
      - arm64

    # GOARM to build for when GOARCH is arm.
    # For more info refer to: https://golang.org/doc/install/source#environment
    # Default is only 6.
    goarm:
      - 6
      - 7

    # List of combinations of GOOS + GOARCH + GOARM to ignore.
    # Default is empty.
    ignore:
      - goos: darwin
        goarch: 386
      - goos: linux
        goarch: arm
        goarm: 7

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    # Default is both hooks empty.
    hooks:
      pre: rice embed-go
      post: ./script.sh
```
