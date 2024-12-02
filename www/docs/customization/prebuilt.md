# Import pre-built binaries

<!-- md:pro -->

It is also possible to import pre-built binaries into the GoReleaser lifecycle.

Reasons you might want to do that include:

- You want to build your binaries in different machines due to CGO
- You want to build using a pre-existing `Makefile` or other tool
- You want to speed up the build by running several builds in parallel in
  different machines

In any case, its pretty easy to do that now:

```yaml title=".goreleaser.yaml"
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
