# Builds

Builds can be customized in multiple ways.
You can specify for which `GOOS`, `GOARCH` and `GOARM` binaries are built
(GoReleaser will generate a matrix of all combinations), and you can change
the name of the binary, flags, environment variables, hooks and more.

Here is a commented `builds` section with all fields specified:

```yaml
# .goreleaser.yaml
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
    #
    # Templates: allowed.
    asmflags:
      - -D mysymbol
      - all=-trimpath={{.Env.GOPATH}}

    # Custom gcflags.
    #
    # Templates: allowed.
    gcflags:
      - all=-trimpath={{.Env.GOPATH}}
      - ./dontoptimizeme=-N

    # Custom ldflags.
    #
    # Default: '-s -w -X main.version={{.Version}} -X main.commit={{.Commit}} -X main.date={{.Date}} -X main.builtBy=goreleaser'.
    # Templates: allowed.
    ldflags:
      - -s -w -X main.build={{.Version}}
      - ./usemsan=-msan

    # Custom Go build mode.
    #
    # Valid options:
    # - `c-shared`
    # - `c-archive`
    # - `pie`
    buildmode: c-shared

    # Custom build tags templates.
    tags:
      - osusergo
      - netgo
      - static_build
      - feature

    # Custom environment variables to be set during the builds.
    # Invalid environment variables will be ignored.
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - CGO_ENABLED=0
      # complex, templated envs (v1.14+):
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CC=o64-clang{{- end }}
          {{- if eq .Arch "arm64"}}CC=aarch64-apple-darwin20.2-clang{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CC=x86_64-w64-mingw32-gcc{{- end }}
        {{- end }}

    # GOOS list to build for.
    # For more info refer to: https://go.dev/doc/install/source#environment
    #
    # Default: [ 'darwin', 'linux', 'windows' ].
    goos:
      - freebsd
      - windows

    # GOARCH to build for.
    # For more info refer to: https://go.dev/doc/install/source#environment
    #
    # Default: [ '386', 'amd64', 'arm64' ].
    goarch:
      - amd64
      - arm
      - arm64

    # GOARM to build for when GOARCH is arm.
    # For more info refer to: https://go.dev/doc/install/source#environment
    #
    # Default: [ 6 ].
    goarm:
      - 6
      - 7

    # GOAMD64 to build when GOARCH is amd64.
    # For more info refer to: https://go.dev/doc/install/source#environment
    #
    # Default: [ 'v1' ].
    goamd64:
      - v2
      - v3

    # GOMIPS and GOMIPS64 to build when GOARCH is mips, mips64, mipsle or mips64le.
    # For more info refer to: https://go.dev/doc/install/source#environment
    #
    # Default: [ 'hardfloat' ].
    gomips:
      - hardfloat
      - softfloat

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
    # applicable: `_{goarm}`, `_{goamd64}`, `_{gomips}`.
    #
    # Special values:
    # - go_118_first_class: evaluates to the first-class ports of go1.18.
    # - go_first_class: evaluates to latest stable go first-class ports,
    #   currently same as 1.18.
    #
    # This overrides `goos`, `goarch`, `goarm`, `gomips`, `goamd64` and
    # `ignores`.
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
    gobinary: "go1.13.4"

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
    # This is a GoReleaser Pro feature.
    # Valid options are: `go` and `prebuilt`.
    #
    # Default: 'go'.
    builder: prebuilt

    # Overrides allows to override some fields for specific targets.
    # This can be specially useful when using CGO.
    # Note: it'll only match if the full target matches.
    overrides:
      - goos: darwin
        goarch: amd64
        goamd64: v1 # Note: if you leave this
        goarm: ""
        gomips: ""
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
```

!!! warning "GOAMD64"

    You usually will need to specify the `goamd64` version beign used, e.g., in
    `overrides`, and other places too.

{% include-markdown "../includes/templates.md" comments=false %}

!!! info

    First-class build targets are gathered by running:
    ```sh
    go tool dist list -json | jq -r '.[] | select(.FirstClass) | [.GOOS, .GOARCH] | @tsv'
    ```
    We also recommend reading the [official wiki about Go ports](https://go.dev/wiki/PortingPolicy#first-class-ports).

Here is an example with multiple binaries:

```yaml
# .goreleaser.yaml
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

The binary name field supports [templating](templates.md). The
following build details are exposed:

<!-- to format the tables, use: https://tabletomarkdown.com/format-markdown-table/ -->

| Key     | Description                       |
| ------- | --------------------------------- |
| .Os     | `GOOS`                            |
| .Arch   | `GOARCH`                          |
| .Arm    | `GOARM`                           |
| .Ext    | Extension, e.g. `.exe`            |
| .Target | Build target, e.g. `darwin_amd64` |

## Passing environment variables to ldflags

You can do that by using `{{ .Env.VARIABLE_NAME }}` in the template, for
example:

```yaml
# .goreleaser.yaml
builds:
  - ldflags:
   - -s -w -X "main.goversion={{.Env.GOVERSION}}"
```

Then you can run:

```sh
GOVERSION=$(go version) goreleaser
```

## Build Hooks

Both pre and post hooks run **for each build target**, regardless of whether
these targets are generated via a matrix of OSes and architectures or defined
explicitly.

In addition to simple declarations as shown above _multiple_ hooks can be
declared to help retaining reusability of config between different build
environments.

```yaml
# .goreleaser.yaml
builds:
  - id: "with-hooks"
    targets:
      - "darwin_amd64"
      - "windows_amd64"
    hooks:
      pre:
        - first-script.sh
        - second-script.sh
      post:
        - upx "{{ .Path }}"
        - codesign -project="{{ .ProjectName }}" "{{ .Path }}"
```

Each hook can also have its own work directory and environment variables:

```yaml
# .goreleaser.yaml
builds:
  - id: "with-hooks"
    targets:
      - "darwin_amd64"
      - "windows_amd64"
    hooks:
      pre:
        - cmd: first-script.sh
          dir:
            "{{ dir .Dist}}"
            # Always print command output, otherwise only visible in debug mode.
          output: true
          env:
            - HOOK_SPECIFIC_VAR={{ .Env.GLOBAL_VAR }}
        - second-script.sh
```

All properties of a hook (`cmd`, `dir` and `env`) support
[templating](templates.md) with `post` hooks having binary artifact
available (as these run _after_ the build).
Additionally the following build details are exposed to both `pre` and `post`
hooks:

<!-- to format the tables, use: https://tabletomarkdown.com/format-markdown-table/ -->

| Key     | Description                            |
| ------- | -------------------------------------- |
| .Name   | Filename of the binary, e.g. `bin.exe` |
| .Ext    | Extension, e.g. `.exe`                 |
| .Path   | Absolute path to the binary            |
| .Target | Build target, e.g. `darwin_amd64`      |

Environment variables are inherited and overridden in the following order:

- global (`env`)
- build (`builds[].env`)
- hook (`builds[].hooks.pre[].env` and `builds[].hooks.post[].env`)

## Go Modules

If you use Go 1.11+ with go modules or vgo, when GoReleaser runs it may try to
download the dependencies. Since several builds run in parallel, it is very
likely to fail.

You can solve this by running `go mod tidy` before calling `goreleaser` or
by adding a [hook][] doing that on your `.goreleaser.yaml` file:

```yaml
# .goreleaser.yaml
before:
  hooks:
    - go mod tidy
# rest of the file...
```

[hook]: hooks.md

## Define Build Tag

GoReleaser uses `git describe` to get the build tag. You can set
a different build tag using the environment variable `GORELEASER_CURRENT_TAG`.
This is useful in scenarios where two tags point to the same commit.

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

## Import pre-built binaries

{% include-markdown "../includes/pro.md" comments=false %}

It is also possible to import pre-built binaries into the GoReleaser lifecycle.

Reasons you might want to do that include:

- You want to build your binaries in different machines due to CGO
- You want to build using a pre-existing `Makefile` or other tool
- You want to speed up the build by running several builds in parallel in
  different machines

In any case, its pretty easy to do that now:

```yaml
# .goreleaser.yaml
builds:
  - # Set the builder to prebuilt
    builder: prebuilt

    # When builder is `prebuilt` there are no defaults for goos, goarch,
    # goarm, gomips, goamd64 and targets, so you always have to specify them:
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    goamd64:
      - v1

    # prebuilt specific options
    prebuilt:
      # Path must be the template path to the binaries.
      # GoReleaser removes the `dist` directory before running, so you will likely
      # want to put the binaries elsewhere.
      # This field is required when using the `prebuilt` builder.
      path: output/mybin_{{ .Os }}_{{ .Arch }}{{ with .Amd64 }}_{{ . }}{{ end }}/mybin

    # Use 'binary' to set the final name of your binary.
    # This is the name that will be used in archives et al.
    binary: bin/mybin
```

!!! tip

    You can think of `prebuilt.path` as being the "external path" and the
    `binary` as being the "internal path to binary".

This example config will import into your release pipeline the following
binaries:

- `output/mybin_linux_amd64_v1`
- `output/mybin_linux_arm64`
- `output/mybin_darwin_amd64_v1`
- `output/mybin_darwin_arm64`

The other steps of the pipeline will act as if those were built by GoReleaser
itself.
There is no difference in how the binaries are handled.

!!! tip

    A cool tip here, specially when using CGO, is that you can have one
    `.goreleaser.yaml` file just for the builds, build each in its own machine
    with [`goreleaser build --single-target`](../cmd/goreleaser_build.md) and
    have a second `.goreleaser.yaml` file that imports those binaries
    and release them.
    This tip can also be used to speed up the build process if you run all the
    builds in different machines in parallel.

!!! warning

    GoReleaser will try to stat the final path, if any error happens while
    doing that (e.g. file does not exist or permission issues),
    GoReleaser will fail.

!!! warning

    When using the `prebuilt` binary, there are no defaults for `goos`,
    `goarch`, `goarm`, `gomips` and `goamd64`.
    You'll need to either provide them or the final `targets` matrix.

If you'd like to see this in action, check [this example on GitHub](https://github.com/caarlos0/goreleaser-pro-prebuilt-example).

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

### Go's first class ports

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

```yaml
# .goreleaser.yaml
builds:
  - id: "my-library"

    # Configure the buildmode flag to output a shared library
    buildmode: "c-shared" # or "c-archive" for a static library
```

## Complex template environment variables

Builds environment variables accept templates.

You can leverage that to have a single build configuration with different
environment variables for each platform, for example.

A common example of this is the variables `CC` and `CXX`.

Here are two different examples:

### Using multiple envs

This example creates once `CC_` and `CXX_` variable for each platform, and then
set `CC` and `CXX` to the right one:

```yaml
# .goreleaser.yml
builds:
  - id: mybin
    binary: mybin
    main: .
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
      - CC_darwin_amd64=o64-clang
      - CXX_darwin_amd64=o64-clang+
      - CC_darwin_arm64=aarch64-apple-darwin20.2-clang
      - CXX_darwin_arm64=aarch64-apple-darwin20.2-clang++
      - CC_windows_amd64=x86_64-w64-mingw32-gc
      - CXX_windows_amd64=x86_64-w64-mingw32-g++
      - 'CC={{ index .Env (print "CC_" .Os "_" .Arch) }}'
      - 'CXX={{ index .Env (print "CXX_" .Os "_" .Arch) }}'
```

### Using `if` statements

This example uses `if` statements to set `CC` and `CXX`:

```yaml
# .goreleaser.yml
builds:
  - id: mybin
    binary: mybin
    main: .
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CC=o64-clang{{- end }}
          {{- if eq .Arch "arm64"}}CC=aarch64-apple-darwin20.2-clang{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CC=x86_64-w64-mingw32-gcc{{- end }}
        {{- end }}
      - >-
        {{- if eq .Os "darwin" }}
          {{- if eq .Arch "amd64"}}CXX=o64-clang+{{- end }}
          {{- if eq .Arch "arm64"}}CXX=aarch64-apple-darwin20.2-clang++{{- end }}
        {{- end }}
        {{- if eq .Os "windows" }}
          {{- if eq .Arch "amd64" }}CXX=x86_64-w64-mingw32-g++{{- end }}
        {{- end }}
```
