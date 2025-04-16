# Rust

<!-- md:version v2.5 -->

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
    # Default: [ "x86_64-unknown-linux-gnu", "x86_64-apple-darwin", "x86_64-pc-windows-gnu", "aarch64-unknown-linux-gnu", "aarch64-apple-darwin" ]
    targets:
      - x86_64-apple-darwin
      - x86_64-pc-windows-gnu

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the cargo build command(s).
    #
    # Default: '.'.
    dir: my-app

    # Set a specific cargo binary to use when building.
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
    # Default: "--release".
    flags:
      - --release
      - -p=subproject # when using cargo-workspaces

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

Some options are not supported yet[^fail], but it should be usable at least for
simple projects already!

!!! tip

    Learn more about [build hooks](./hooks.md).

GoReleaser will run `rustup target add` for each defined target.
You can use before hooks to install `cargo-zigbuild`.
If you want to use `cargo-cross` instead, you can make sure it is installed and
then make few changes:

```yaml title=".goreleaser.yaml"
builds:
  - # Use cargo cross:
    builder: rust
    tool: cross
    command: build
    targets:
      - x86_64-apple-darwin
      - x86_64-pc-windows-gnu
```

## Publishing with Cargo

You can use [global after hooks](../hooks.md) to do it:

```yaml title=".goreleaser.yaml"
# global after hooks
after:
  - cmd: "cargo publish {{ if .IsSnapshot }}--dry-run{{ end }} --quiet --no-verify"
```

## Caveats

### Targets

GoReleaser will translate Rust's Os/Arch triple into a GOOS/GOARCH pair, so
templates should work the same as before.
The original target name is available in templates as `.Target`, and so are
`.Vendor` and `.Environment`.

### Environment setup

GoReleaser will not install Cargo, Rustup, Zig, or cargo-zigbuild for you.
Make sure to install them before running GoReleaser.

Remember that you may also need to run `rustup default stable`.

GoReleaser **will**, however, run `rustup target add` for each target you
declare.

You can also add them to your [global before hooks](../hooks.md), e.g.:

```yaml title=".goreleaser.yaml"
before:
  hooks:
    - rustup default stable
    - cargo install --locked cargo-zigbuild
```

### Cargo Workspaces

Projects that use Cargo workspaces might not work depending on usage.
If you want to try it, add `-p=[name]` to the `flags` property.
We might improve this in the future.

[^fail]:
    GoReleaser will error if you try to use them. Give it a try with
    `goreleaser r --snapshot --clean`.

<!-- md:templates -->
