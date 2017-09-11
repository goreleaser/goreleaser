---
title: Custom builds
---

Builds can be customized in multiple ways. You can change which `GOOS` and
`GOARCH` would be built, the name of the binary, flags, ldflags, hooks and etc.

Here is a full commented `builds` section:

```yml
# .goreleaser.yml
builds:
  # You can have multiple builds defined as a common yaml list
  -
    # Path to main.go file or main package.
    # Default is `.`
    main: ./cmd/main.go

    # Name of the binary.
    # Default is the name of the project directory.
    binary: program

    # Custom build tags.
    # Default is empty
    flags: -tags dev

    # Custom ldflags template.
    # This is parsed with Golang template engine and the following variables
    # are available:
    # - Date
    # - Commit
    # - Tag
    # - Version (Tag with the `v` prefix stripped)
    # The default is `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}}`
    # Date format is `2006-01-02_15:04:05`
    ldflags: -s -w -X main.build={{.Version}}

    # Custom environment variables to be set durign the builds.
    # Default is empty
    env:
      - CGO_ENABLED=0

    # GOOS list to build in.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are darwin and linux
    goos:
      - freebsd
      - windows

    # GOARCH to build in.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are 386 and amd64
    goarch:
      - amd64
      - arm
      - arm64

    # GOARM to build in when GOARCH is arm.
    # For more info refer to https://golang.org/doc/install/source#environment
    # Defaults are 6
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

    # Hooks can be used to customize the final binary, for example, to run
    # generator or whatever you want.
    # Default is both hooks empty.
    hooks:
      pre: rice embed-go
      post: ./script.sh
```
