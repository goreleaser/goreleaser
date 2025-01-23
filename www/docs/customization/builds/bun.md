# Bun

<!-- md:version v2.6 -->

<!-- md:alpha -->

You can now build TypeScript binaries using `bun build --compile` and GoReleaser!

Simply set the `builder` to `bun`, for instance:

```yaml title=".goreleaser.yaml"
builds:
  # You can have multiple builds defined as a yaml list
  - #
    # ID of the build.
    #
    # Default: Project directory name.
    id: "my-build"

    # Use bun.
    builder: bun

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # List of targets to be built, in Bun's format.
    #
    # See: https://bun.sh/docs/bundler/executables
    # The `bun-` prefix is added automatically.
    # Default: [ "linux-x64-modern", "linux-arm64", "darwin-x64", "darwin-arm64", "windows-x64-modern" ]
    targets:
      - linux-x64-modern
      - darwin-arm64

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the `bun build` command(s).
    #
    # Default: '.'.
    dir: my-app

    # Main entry point.
    #
    # Default: extracted from package.json or `.`.
    main: "file.ts"

    # Set a specific bun binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "bun".
    # Templates: allowed.
    tool: "bun-nightly"

    # Sets the command to run to build.
    #
    # Default: build.
    command: not-build

    # Custom flags.
    #
    # Templates: allowed.
    # Default: ["--compile"].
    flags:
      - --minify

    # Custom environment variables to be set during the builds.
    # Invalid environment variables will be ignored.
    #
    # Default: os.Environ() ++ env config section.
    # Templates: allowed.
    env:
      - FOO=bar

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    #
    # Templates: allowed.
    hooks:
      pre: ./foo.sh
      post: ./script.sh {{ .Path }}

    # If true, skip the build.
    # Useful for library projects.
    skip: false
```

Some options are not supported yet[^fail], but it should be usable for
most projects already!

!!! tip

    Learn more about [build hooks](./hooks.md).

### Environment setup

GoReleaser will not install Bun or any other dependencies on which your
workflow depends. Make sure to install them before running GoReleaser.

## Caveats

GoReleaser will translate Bun's Os/Arch pair into a GOOS/GOARCH pair, so
templates should work the same as before.
The original target name is available in templates as `.Target`, and so is the
Modern/Baseline bit as `.Type`.

[^fail]:
    GoReleaser will error if you try to use them. Give it a try with
    `goreleaser r --snapshot --clean`.

<!-- md:templates -->
