# Builds

Builds can be customized in multiple ways.
You can specify for which `GOOS`, `GOARCH` and `GOARM` binaries are built
(goreleaser will generate a matrix of all combinations), and you can change
the name of the binary, flags, environment variables, hooks and etc.

Here is a commented `builds` section with all fields specified:

```yaml
# .goreleaser.yaml
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
    # Notice: when used with `gomod.proxy`, this must be a package.
    #
    # Default is `.`.
    main: ./cmd/my-app

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

    # Custom build tags templates.
    # Default is empty.
    tags:
      - osusergo
      - netgo
      - static_build
      - feature

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
    # Defaults are 386, amd64 and arm64.
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
    # Default is only hardfloat.
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

    # Optionally override the matrix generation and specify only the final list of targets.
    # Format is `{goos}_{goarch}` with optionally a suffix with `_{goarm}` or `_{gomips}`.
    # This overrides `goos`, `goarch`, `goarm`, `gomips` and `ignores`.
    targets:
      - linux_amd64
      - darwin_arm64
      - linux_arm_6

    # Set a specific go binary to use when building. It is safe to ignore
    # this option in most cases.
    # Default is "go"
    gobinary: "go1.13.4"

    # Set the modified timestamp on the output binary, typically
    # you would do this to ensure a build was reproducible. Pass
    # empty string to skip modifying the output.
    # Default is empty string.
    mod_timestamp: '{{ .CommitTimestamp }}'

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    # Those fields allow templates.
    # Default is both hooks empty.
    hooks:
      pre: rice embed-go
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    # Useful for library projects.
    # Default is false
    skip: false

    # By default, GoRelaser will create your binaries inside `dist/${BuildID}_${BuildTarget}`, which is an unique directory per build target in the matrix.
    # You are able to set subdirs within that folder using the `binary` property.
    #
    # However, if for some reason you don't want that unique directory to be created, you can set this property.
    # If you do, you are responsible of keeping different builds from overriding each other.
    #
    # Defaults to `false`.
    no_unique_dist_dir: true

    # Builder allows you to use a different build implementation.
    # This is a GoReleaser Pro feature.
    # Valid options are: `go` and `prebuilt`.
    # Defaults to `go`.
    builder: prebuilt
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

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

The binary name field supports [templating](/customization/templates/). The following build details are exposed:

| Key     | Description                      |
|---------|----------------------------------|
| .Os     | `GOOS`                           |
| .Arch   | `GOARCH`                         |
| .Arm    | `GOARM`                          |
| .Ext    | Extension, e.g. `.exe`           |
| .Target | Build target, e.g. `darwin_amd64`|

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
these targets are generated via a matrix of OSes and architectures
or defined explicitly.

In addition to simple declarations as shown above _multiple_ hooks can be declared
to help retaining reusability of config between different build environments.

```yaml
# .goreleaser.yaml
builds:
  -
    id: "with-hooks"
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
  -
    id: "with-hooks"
    targets:
     - "darwin_amd64"
     - "windows_amd64"
    hooks:
      pre:
       - cmd: first-script.sh
         dir: "{{ dir .Dist}}"
         env:
          - HOOK_SPECIFIC_VAR={{ .Env.GLOBAL_VAR }}
       - second-script.sh
```

All properties of a hook (`cmd`, `dir` and `env`) support [templating](/customization/templates/)
with `post` hooks having binary artifact available (as these run _after_ the build).
Additionally the following build details are exposed to both `pre` and `post` hooks:

| Key     | Description                            |
|---------|----------------------------------------|
| .Name   | Filename of the binary, e.g. `bin.exe` |
| .Ext    | Extension, e.g. `.exe`                 |
| .Path   | Absolute path to the binary            |
| .Target | Build target, e.g. `darwin_amd64`      |

Environment variables are inherited and overridden in the following order:

 - global (`env`)
 - build (`builds[].env`)
 - hook (`builds[].hooks.pre[].env` and `builds[].hooks.post[].env`)

## Go Modules

 If you use Go 1.11+ with go modules or vgo, when GoReleaser runs it may
 try to download the dependencies. Since several builds run in parallel, it is
 very likely to fail.

 You can solve this by running `go mod tidy` before calling `goreleaser` or
 by adding a [hook][] doing that on your `.goreleaser.yaml` file:

 ```yaml
 # .goreleaser.yaml
 before:
   hooks:
   - go mod tidy
 # rest of the file...
 ```

 [hook]: /customization/hooks

## Define Build Tag

GoReleaser uses `git describe` to get the build tag. You can set
a different build tag using the environment variable `GORELEASER_CURRENT_TAG`.
This is useful in scenarios where two tags point to the same commit.

## Reproducible Builds

To make your releases, checksums, and signatures reproducible, you will need to make some (if not all) of the following modifications to the build defaults in GoReleaser:

* Modify `ldflags`: by default `main.Date` is set to the time GoReleaser is run (`{{.Date}}`), you can set this to `{{.CommitDate}}` or just not pass the variable.
* Modify `mod_timestamp`: by default this is empty string, set to `{{.CommitTimestamp}}` or a constant value instead.
* If you do not run your builds from a consistent directory structure, pass `-trimpath` to `flags`.
* Remove uses of the `time` template function. This function returns a new value on every call and is not deterministic.

## Import pre-built binaries

!!! success "GoReleaser Pro"
    The prebuilt builder is a [GoReleaser Pro feature](/pro/).

Since GoReleaser Pro v0.179.0, it is possible to import pre-built binaries into the GoReleaser lifecycle.

Reasons you might want to do that include:

- You want to build your binaries in different machines due to CGO
- You want to build using a pre-existing `Makefile` or other tool
- You want to speed up the build by running several builds in parallel in different machines

In any case, its pretty easy to do that now:

```yaml
# .goreleaser.yaml
builds:
-
  # Set the builder to prebuilt
  builder: prebuilt

  # When builder is `prebuilt` there are no defaults for goos, goarch,
  # goarm, gomips and targets.
  goos:
  - linux
  - darwin
  goarch:
  - amd64
  - arm64

  # prebuilt specific options
  prebuilt:
    # Path must be the template path to the binaries.
    # GoReleaser removes the `dist` folder before running, so you will likely
    # want to put the binaries elsewhere.
    # This field is required when using the `prebuilt` builder.
    path: output/mybin_{{ .Os }}_{{ .Arch }}
```

This example config will import into your release pipeline the following binaries:

- `output/mybin_linux_amd64`
- `output/mybin_linux_arm64`
- `output/mybin_darwin_amd64`
- `output/mybin_darwin_arm64`

The other steps of the pipeline will act as if those were built by GoReleaser itself.
There is no difference in how the binaries are handled.

!!! tip
    A cool tip here, specially when using CGO, is that you can have one
    `.goreleaser.yaml` file just for the builds, build each in its own machine
    with [`goreleaser build --single-target`](/cmd/goreleaser_build/) and
    have a second `.goreleaser.yaml` file that imports those binaries
    and release them.
    This tip can also be used to speed up the build process if you run all the
    builds in different machines in parallel.

!!! warning
    GoReleaser will try to stat the final path, if any error happens while
    doing that (e.g. file does not exist or permission issues),
    GoReleaser will fail.

!!! warning
    When using the `prebuilt` binary, there are no defaults for `goos` et al,
    so you need to either provide those or the final `targets` matrix.
