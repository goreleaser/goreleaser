# UV

<!-- md:version v2.9 -->

You can now build Python `wheel` and `sdist` files using `uv build` and
GoReleaser!

Simply set the `builder` to `uv` and set the `buildmode` you want:

```yaml title=".goreleaser.yaml"
builds:
  - builder: uv
    buildmode: wheel
```

The `.whl` and `.tar.gz` files can then be signed, checksummed, used inside
Docker images, and more.

## Options

```yaml title=".goreleaser.yaml"
builds:
  # You can have multiple builds defined as a yaml list
  - #
    # ID of the build.
    #
    # Default: Project directory name.
    id: "my-build"

    # Use uv.
    builder: uv

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the uv build command(s).
    #
    # Default: ".".
    dir: my-app

    # The build mode.
    #
    # Valid options: "wheel", "sdist".
    # Default: "wheel".
    buildmode: sdist

    # Set a specific uv binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "uv".
    # Templates: allowed.
    tool: uv

    # Sets the command to run to build.
    # Can be useful if you want to build tests, for example,
    # in which case you can set this to "test".
    # It is safe to ignore this option in most cases.
    #
    # Default: build.
    command: build

    # Custom flags.
    #
    # Templates: allowed.
    flags:
      - --offline

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
```

!!! warning

    At this time only the target `py3-none-any` is supported.

## Building both wheel and sdist

You need to declare 2 builds, one for each mode:

```yaml title=".goreleaser.yaml"
builds:
  - id: wheel
    builder: uv
    buildmode: wheel
  - id: sdist
    builder: uv
    buildmode: sdist
```

## Publishing to PyPi

You can use [global after hooks](../hooks.md) to do it:

```yaml title=".goreleaser.yaml"
# global after hooks
after:
  - cmd: "uv publish"
    if: "{{ .IsRelease }}"
```
