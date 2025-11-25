# Deno

<!-- md:version v2.6 -->

You can now build TypeScript binaries using `deno compile` and GoReleaser!

Simply set the `builder` to `deno`, for instance:

```yaml title=".goreleaser.yaml"
builds:
  # You can have multiple builds defined as a yaml list
  - #
    # ID of the build.
    #
    # Default: Project directory name.
    id: "my-build"

    # Use deno.
    builder: deno

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # List of targets to be built, in Deno's format.
    #
    # See: https://docs.deno.com/runtime/reference/cli/compile/#supported-targets
    # Default: [ "x86_64-pc-windows-msvc", "x86_64-apple-darwin", "aarch64-apple-darwin", "x86_64-unknown-linux-gnu", "aarch64-unknown-linux-gnu" ]
    targets:
      - x86_64-unknown-linux-gnu
      - aarch64-unknown-linux-gnu
      - x86_64-pc-windows-msvc
      - x86_64-apple-darwin
      - aarch64-apple-darwin

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the `deno compile` command(s).
    #
    # Default: '.'.
    dir: my-app

    # Main entry point.
    #
    # Default: 'main.ts'.
    main: "file.ts"

    # Set a specific deno binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: 'deno'.
    # Templates: allowed.
    tool: "deno-canary"

    # Sets the command to run to build.
    #
    # Default: 'compile'.
    command: not-build

    # Custom flags.
    #
    # Templates: allowed.
    # Default: [].
    flags:
      - --allow-read
      - --allow-net

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

GoReleaser will not install Deno or any other dependencies on which your
workflow depends. Make sure to install them before running GoReleaser.

## Caveats

GoReleaser will translate Deno's Os/Arch pair into a GOOS/GOARCH pair, so
templates should work the same as before.
The original target name is available in templates as `.Target`, and so is the
the ABI and Vendor as `.Abi` and `.Vendor`, respectively.

[^fail]:
    GoReleaser will error if you try to use them. Give it a try with
    `goreleaser r --snapshot --clean`.

<!-- md:templates -->
