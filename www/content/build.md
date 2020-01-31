---
title: Builds
series: customization
hideFromIndex: true
weight: 30
---

Builds can be customized in multiple ways.
You can specify for which `GOOS`, `GOARCH` and `GOARM` binaries are built
(goreleaser will generate a matrix of all combinations), and you can changed
the name of the binary, flags, environment variables, hooks and etc.

Here is a commented `builds` section with all fields specified:

```yml
# .goreleaser.yml
builds:
  # You can have multiple builds defined as a yaml list
  -
    # ID of the build.
    # Defaults to the project name.
    id: "my-build"

    # Path to project's (sub)directory containing Go code.
    # This is the working directory for the Go build command(s).
    # Default is `.`.
    dir: go

    # Path to main.go file or main package.
    # Default is `.`.
    main: ./cmd/main.go

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    # Default is the name of the project directory.
    binary: program

    # Custom flags templates.
    # Default is empty.
    flags:
      - -tags=dev
      - -v

    # Custom asmflags templates.
    # Default is empty.
    asmflags:
      - -D mysymbol
      - all=-trimpath={{.Env.GOPATH}}

    # Custom gcflags templates.
    # Default is empty.
    gcflags:
      - all=-trimpath={{.Env.GOPATH}}
      - ./dontoptimizeme=-N

    # Custom ldflags templates.
    # Default is `-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser`.
    ldflags:
     - -s -w -X main.build={{.Version}}
     - ./usemsan=-msan

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

    # GOMIPS and GOMIPS64 to build when GOARCH is mips, mips64, mipsle or mips64le.
    # For more info refer to: https://golang.org/doc/install/source#environment
    # Default is empty.
    gomips:
      - hardfloat
      - softfloat

    # List of combinations of GOOS + GOARCH + GOARM to ignore.
    # Default is empty.
    ignore:
      - goos: darwin
        goarch: 386
      - goos: linux
        goarch: arm
        goarm: 7
      - goarm: mips64
        gomips: hardfloat

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    # Those fields allow templates.
    # Default is both hooks empty.
    hooks:
      pre: rice embed-go
      post: ./script.sh
```

> Learn more about the [name template engine](/templates).

## Passing environment variables to ldflags

You can do that by using `{{ .Env.VARIABLE_NAME }}` in the template, for
example:

```yaml
builds:
  - ldflags:
   - -s -w -X "main.goversion={{.Env.GOVERSION}}"
```

Then you can run:

```sh
GOVERSION=$(go version) goreleaser
```

## Go Modules

 If you use Go 1.11+ with go modules or vgo, when GoReleaser runs it may
 try to download the dependencies. Since several builds run in parallel, it is
 very likely to fail.

 You can solve this by running `go mod tidy` before calling `goreleaser` or
 by adding a [hook][] doing that on your `.goreleaser.yaml` file:

 ```yaml
 before:
   hooks:
   - go mod tidy
 # rest of the file...
 ```

 [hook]: /hooks

## Define Build Tag

GoReleaser uses `git describe` to get the build tag. You can set
a different build tag using the environment variable `GORELEASER_CURRENT_TAG`.
This is useful in scenarios where two tags point to the same commit.
