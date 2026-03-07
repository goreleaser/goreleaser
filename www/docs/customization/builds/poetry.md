# Poetry

<!-- md:version v2.9 -->

You can now build Python `wheel` and `sdist` files using `poetry build` and
GoReleaser!

Simply set the `builder` to `poetry` and set the `buildmode` you want:

```yaml title=".goreleaser.yaml"
builds:
  - builder: poetry
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

    # Use poetry.
    builder: poetry

    # Path to project's (sub)directory containing the code.
    # This is the working directory for the poetry build command(s).
    #
    # Default: ".".
    dir: my-app

    # The build mode.
    #
    # Valid options: "wheel", "sdist".
    # Default: "wheel".
    buildmode: sdist

    # Set a specific poetry binary to use when building.
    # It is safe to ignore this option in most cases.
    #
    # Default: "poetry".
    # Templates: allowed.
    tool: poetry

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
      - --no-cache

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
    builder: poetry
    buildmode: wheel
  - id: sdist
    builder: poetry
    buildmode: sdist
```

## Publishing to PyPi

You can use [global after hooks](../hooks.md) to do it:

```yaml title=".goreleaser.yaml"
# global after hooks
after:
  - cmd: "poetry publish"
    if: "{{ .IsRelease }}"
```
