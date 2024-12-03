# Builds (Rust)

<!-- md:version v2.5-unreleased -->

<!-- md:alpha -->

You can now build Rust binaries using `cargo zigbuild` and GoReleaser!

Simply set the `builder` to `rust`, for instance:

```yaml title=".goreleaser.yaml"
builds:
  # You can have multiple builds defined as a yaml list
  - #
    # ID of the build.
    #
    # Default: Project directory name.
    id: "my-build"

    # Use rust.
    builder: rust

    # Binary name.
    # Can be a path (e.g. `bin/app`) to wrap the binary in a directory.
    #
    # Default: Project directory name.
    binary: program

    # List of targets to be built, in Rust's format.
    targets:
      - x86_64-apple-darwin
      - x86_64-pc-windows-gnu

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the Zig build command(s).
    #
    # Default: '.'.
    dir: my-app

    # Set a specific zig binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "cargo".
    # Templates: allowed.
    tool: "cross"

    # Sets the command to run to build.
    # Can be useful if you want to build tests, for example,
    # in which case you can set this to "test".
    # It is safe to ignore this option in most cases.
    #
    # Default: zigbuild.
    command: build

    # Custom flags.
    #
    # Templates: allowed.
    flags:
      - --release
```

Some options are not supported yet[^fail], but it should be usable at least for
simple projects already!

GoReleaser will run `rustup target add` for each defined target.
You can use before hooks to install `cargo-zigbuild`.
If you want to use `cargo-cross` instead, you can make sure it is installed and
then make few changes:

```yaml title=".goreleaser.yaml"
builds:
  - # Use Rust zigbuild
    builder: rust
    tool: cross
    command: build
    targets:
      - x86_64-apple-darwin
      - x86_64-pc-windows-gnu
```

## Caveats

GoReleaser will translate Rust's Os/Arch triple into a GOOS/GOARCH pair, so
templates should work the same as before.
The original target name is available in templates as `.Target`, and so are
`.Vendor` and `.Environment`.

[^fail]:
    GoReleaser will error if you try to use them. Give it a try with
    `goreleaser r --snapshot --clean`.
