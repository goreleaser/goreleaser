# Go

Builds can be customized in multiple ways.

You can specify for which `GOOS`, `GOARCH` and `GOARM` binaries are built
(GoReleaser will generate a matrix of all combinations), and you can change
the name of the binary, flags, environment variables, hooks and more.

Here is a commented `builds` section with all fields specified:

```yaml title=".goreleaser.yaml"
builds:
  # You can have multiple builds defined as a yaml list
  - #
    # ID of the build.
    #
    # Default: Project directory name.
    id: "my-build"

    # Path to main.go file or main package.
    # Notice: when used with `gomod.proxy`, this must be a package.
    #
    # Default: `.`.
    main: ./cmd/my-app

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # Custom flags.
    #
    # Templates: allowed.
    flags:
      - -tags=dev
      - -v

    # Custom asmflags.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies
    # and https://pkg.go.dev/cmd/asm
    #
    # Templates: allowed.
    asmflags:
      - -D mysymbol
      - all=-trimpath={{.Env.GOPATH}}

    # Custom gcflags.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies
    # and https://pkg.go.dev/cmd/compile
    #
    # Templates: allowed.
    gcflags:
      - all=-trimpath={{.Env.GOPATH}}
      - ./dontoptimizeme=-N

    # Custom ldflags.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Compile_packages_and_dependencies
    # and https://pkg.go.dev/cmd/link
    #
    # Default: '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser'.
    # Templates: allowed.
    ldflags:
      - -s -w -X main.build={{.Version}}
      - ./usemsan=-msan

    # Custom Go build mode.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Build_modes
    #
    # Valid options:
    # - `c-shared`
    # - `c-archive`
    # - `pie`
    buildmode: c-shared

    # Custom build tags templates.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Build_constraints
    tags:
      - osusergo
      - netgo
      - static_build
      - feature

    # Custom environment variables to be set during the builds.
    # Invalid environment variables will be ignored.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - CGO_ENABLED=0
      # complex, templated envs:
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CC=o64-clang{{- end }}
          {{- if eq .Arch "arm64"}}CC=aarch64-apple-darwin20.2-clang{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CC=x86_64-w64-mingw32-gcc{{- end }}
        {{- end }}

    # GOOS list to build for.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    #
    # Default: [ 'darwin', 'linux', 'windows' ].
    goos:
      - freebsd
      - windows

    # GOARCH to build for.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    #
    # Default: [ '386', 'amd64', 'arm64' ].
    goarch:
      - amd64
      - arm
      - arm64

    # GOARM to build for when GOARCH is arm.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 6 ].
    goarm:
      - 6
      - 7

    # GOAMD64 to build when GOARCH is amd64.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'v1' ].
    goamd64:
      - v2
      - v3

    # GOARM64 to build when GOARCH is arm64.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'v8.0' ].
    # <!-- md:inline_version v2.4 -->.
    goarm64:
      - v9.0

    # GOMIPS and GOMIPS64 to build when GOARCH is mips, mips64, mipsle or mips64le.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'hardfloat' ].
    # <!-- md:inline_version v2.4 -->.
    gomips:
      - hardfloat
      - softfloat

    # GO386 to build when GOARCH is 386.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'sse2' ].
    # <!-- md:inline_version v2.4 -->.
    go386:
      - sse2
      - softfloat

    # GOPPC64 to build when GOARCH is PPC64.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'power8' ].
    # <!-- md:inline_version v2.4 -->.
    goppc64:
      - power8
      - power9

    # GORISCV64 to build when GOARCH is RISCV64.
    # For more info refer to: https://pkg.go.dev/cmd/go#hdr-Environment_variables
    # and https://go.dev/wiki/MinimumRequirements#microarchitecture-support
    #
    # Default: [ 'rva20u64' ].
    # <!-- md:inline_version v2.4 -->.
    goriscv64:
      - rva22u64

    # List of combinations of GOOS + GOARCH + GOARM to ignore.
    ignore:
      - goos: darwin
        goarch: 386
      - goos: linux
        goarch: arm
        goarm: 7
      - goarm: mips64
      - gomips: hardfloat
      - goamd64: v4

    # Optionally override the matrix generation and specify only the final list
    # of targets.
    #
    # Format is `{goos}_{goarch}` with their respective suffixes when
    # applicable: `_{goarm}`, `_{goamd64}`, `_{gomips}`, `_{go386}`,
    #             `_{goriscv64}`, `_{goarm64}`, `_{goppc64}`.
    #
    # Special values:
    # - go_118_first_class: evaluates to the first-class ports of go1.18.
    # - go_first_class: evaluates to latest stable go first-class ports,
    #   currently same as 1.18.
    #
    # This overrides `goos`, `goarch`, `goarm`, `gomips`, `goamd64`, `go386`,
    #                `goriscv64`, `goarm64`, `goppc64`, and `ignores`.
    targets:
      - go_first_class
      - go_118_first_class
      - linux_amd64_v1
      - darwin_arm64
      - linux_arm_6

    # Set a specific go binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "go".
    # Templates: allowed.
    # <!-- md:inline_version v2.5 -->.
    tool: "go1.13.4"

    # Sets the command to run to build.
    # Can be useful if you want to build tests, for example,
    # in which case you can set this to "test".
    # It is safe to ignore this option in most cases.
    #
    # Default: build.
    command: test

    # Set the modified timestamp on the output binary, typically
    # you would do this to ensure a build was reproducible.
    # Pass an empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    #
    # Templates: allowed.
    hooks:
      pre: rice embed-go
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    # Useful for library projects.
    #
    # Templates: allowed (<!-- md:inline_version v2.3 -->).
    skip: false

    # By default, GoReleaser will create your binaries inside
    # `dist/${BuildID}_${BuildTarget}`, which is a unique directory per build
    # target in the matrix.
    # You can set subdirs within that directory using the `binary` property.
    #
    # However, if for some reason you don't want that unique directory to be
    # created, you can set this property.
    # If you do, you are responsible for keeping different builds from
    # overriding each other.
    #
    # Templates: allowed (<!-- md:inline_version v2.3 -->).
    no_unique_dist_dir: true

    # By default, GoReleaser will check if the main filepath has a main
    # function.
    # This can be used to skip that check, in case you're building tests, for
    # example.
    no_main_check: true

    # Path to project's (sub)directory containing Go code.
    # This is the working directory for the Go build command(s).
    # If dir does not contain a `go.mod` file, and you are using `gomod.proxy`,
    # produced binaries will be invalid.
    # You would likely want to use `main` instead of this.
    #
    # Default: '.'.
    dir: go

    # Builder allows you to use a different build implementation.
    # Valid options are: `go`, `rust`, `zig`, and `prebuilt` (pro-only).
    #
    # Default: 'go'.
    builder: prebuilt

    # Overrides allows to override some fields for specific targets.
    # This can be specially useful when using CGO.
    #
    # Attention: you need to set at least goos and goarch, otherwise it won't
    # match anything.
    overrides:
      - goos: darwin
        goarch: amd64
        goamd64: v1
        goarm: ""
        goarm64: ""
        gomips: ""
        go386: ""
        goriscv64: ""
        goppc64: ""
        ldflags:
          - foo
        tags:
          - bar
        asmflags:
          - foobar
        gcflags:
          - foobaz
        env:
          - CGO_ENABLED=1

    # Set a specific go binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "go".
    # Templates: allowed.
    # Deprecated: use `tool` instead.
    gobinary: "go1.13.4"
```

!!! tip

    Learn more about [build hooks](./hooks.md).

!!! warning "GOAMD64, GORISCV64, GOPPC64, GO386, GOARM, GOARM64"

    You usually will need to specify the complete target in places like
    `targets` and `overrides`.
    This includes the `_{goamd64}` suffix, as well as the other
    GOARCH-specific values.

<!-- md:templates -->

!!! info

    First-class build targets are gathered by running:
    ```sh
    go tool dist list -json | jq -r '.[] | select(.FirstClass) | [.GOOS, .GOARCH] | @tsv'
    ```
    We also recommend reading the [official wiki about Go ports](https://go.dev/wiki/PortingPolicy#first-class-ports).

Here is an example with multiple binaries:

```yaml title=".goreleaser.yaml"
builds:
  - main: ./cmd/cli
    id: "cli"
    binary: cli
    goos:
      - linux
      - darwin
      - windows

  - main: ./cmd/worker
    id: "worker"
    binary: worker
    goos:
      - linux
      - darwin
      - windows

  - main: ./cmd/tracker
    id: "tracker"
    binary: tracker
    goos:
      - linux
      - darwin
      - windows
```

The binary name field supports [templating](../templates.md). The
following build details are exposed:

| Key     | Description                       |
| ------- | --------------------------------- |
| .Os     | `GOOS`                            |
| .Arch   | `GOARCH`                          |
| .Arm    | `GOARM`                           |
| .Ext    | Extension, e.g. `.exe`            |
| .Target | Build target, e.g. `darwin_amd64` |

## Understanding `GOMAXPROCS` in GoReleaser

### `GOMAXPROCS` for GoReleaser

GoReleaser uses
[`automaxprocs`](https://pkg.go.dev/go.uber.org/automaxprocs/maxprocs) to
automatically set `GOMAXPROCS` based on available CPUs, including honoring
container CPU limits.
This determines the number of threads GoReleaser itself uses internally.

GoReleaser also provides a `--parallelism` flag to control how many internal
tasks (e.g., builds, archives, uploads) run concurrently.
If `--parallelism` is not set, GoReleaser defaults to the current value of
`GOMAXPROCS`.

### `GOMAXPROCS` for `go build` commands

Each `go build` command launched by GoReleaser inherits the environment,
including `GOMAXPROCS`.
If `GOMAXPROCS` is not explicitly set, Go will default to the number of **host**
CPUs.

If you're running inside a container and want to respect CPU limits during
builds, you must set `GOMAXPROCS` manually.

### Example

If you want GoReleaser to run up to 10 tasks in parallel, but restrict `go
build` to use only 2 threads:

```sh
GOMAXPROCS=2 goreleaser release --parallelism=10
```

This configures:

- GoReleaser to run up to 10 internal tasks concurrently
- `go build` subprocesses to use only 2 OS threads

## Passing environment variables to ldflags

You can do that by using `{{ .Env.VARIABLE_NAME }}` in the template, for
example:

```yaml title=".goreleaser.yaml"
builds:
  - ldflags:
   - -s -w -X "main.goversion={{.Env.GOVERSION}}"
```

Then you can run:

```sh
GOVERSION=$(go version) goreleaser
```

## Go Modules

If you use Go 1.11+ with go modules or vgo, when GoReleaser runs it may try to
download the dependencies. Since several builds run in parallel, it is very
likely to fail.

You can solve this by running `go mod tidy` before calling `goreleaser` or
by adding a [hook][] doing that on your `.goreleaser.yaml` file:

```yaml title=".goreleaser.yaml"
before:
  hooks:
    - go mod tidy
# rest of the file...
```

[hook]: ../hooks.md

## Reproducible Builds

To make your releases, checksums and signatures reproducible, you will need to
make some (if not all) of the following modifications to the build defaults in
GoReleaser:

- Modify `ldflags`: by default `main.Date` is set to the time GoReleaser is run
  (`{{.Date}}`), you can set this to `{{.CommitDate}}` or just not pass the
  variable.
- Modify `mod_timestamp`: by default this is empty string — which means it'll be
  the compilation time, set to `{{.CommitTimestamp}}` or a constant value
  instead.
- If you do not run your builds from a consistent directory structure, pass
  `-trimpath` to `flags`.
- Remove uses of the `time` template function. This function returns a new value
  on every call and is not deterministic.

## A note about directory names inside `dist`

By default, GoReleaser will create your binaries inside
`dist/${BuildID}_${BuildTarget}`, which is a unique directory per build target
in the matrix.

Those names have no guarantees of remaining the same from one version to
another. If you really need to access them from outside GoReleaser, you should
be able to consistently get the path of a binary by parsing
`dist/artifacts.json`.

You can also set `builds.no_unique_dist_dir` (as documented earlier in this
page), but in that case you are responsible for preventing name conflicts.

### Why is there a `_v1` suffix on `amd64` builds?

Go 1.18 introduced the `GOAMD64` option, and `v1` is the default value for that
option.

Since you can have GoReleaser build for multiple different `GOAMD64` targets, it
adds that suffix to prevent name conflicts. The same thing happens for `arm` and
`GOARM`, `mips` and `GOMIPS` and others.

## Go's first class ports

The `targets` option can take a `go_first_class` special value as target, which
will evaluate to the list of first class ports as defined in the Go wiki.

You can read more about it
[here](https://go.dev/wiki/PortingPolicy#first-class-ports).

## Building shared or static libraries

GoReleaser supports compiling and releasing C shared or static libraries, by
configuring the [Go build mode](https://pkg.go.dev/cmd/go#hdr-Build_modes).

This can be set with `buildmode` in your build.
It now supports `c-shared` and `c-archive`. Other values will transparently be
applied to the build line (via the `-buildmode` flag), but GoReleaser will not
attempt to configure any additional logic.

GoReleaser will:

- set the correct file extension for the target OS.
- package the generated header file (`.h`) in the release bundle.

Example usage:

```yaml title=".goreleaser.yaml"
builds:
  - id: "my-library"

    # Configure the buildmode flag to output a shared library
    buildmode: "c-shared" # or "c-archive" for a static library
```

<!-- md:templates -->
