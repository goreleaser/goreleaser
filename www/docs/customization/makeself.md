# Self-Extracting Archives

GoReleaser can create self-extracting archives using [Makeself][].
These files are executables that self-extract themselves, and might run the
underlying binary, install it, or do other operations.

This is particularly useful for distributing software that needs to be easily
installable without requiring users to manually extract archives.
Typically this supports Linux, MacOS and any other platform that Makeself runs
on.

!!! note

    This feature requires the `makeself` command to be available in
    your system `$PATH`.
    You can install it from your system package manager or from
    [the Makeself project][Makeself].

## Configuration

Here is a commented `makeselfs` section with all fields specified:

```yaml title=".goreleaser.yaml"
makeselfs:
  - #
    # ID of this Makeself package.
    #
    # Default: 'default'.
    id: my-installer

    # IDs of the builds which should be packaged in this Makeself archive.
    #
    # Default: empty (include all).
    ids:
      - my-binary

    # Which OSes to create the packages for.
    #
    # Default: [linux darwin].
    goos:
      - linux
      - darwin

    # Which architectures to create the packages for.
    #
    # Default: empty (all architectures).
    goarch:
      - arm64
      - amd64

    # Name of the package.
    #
    # Default: '{{ .ProjectName }}'.
    # Templates: allowed.
    name: my-package

    # Description of your package.
    #
    # Default: inferred from global metadata.
    # Templates: allowed.
    description: foo bar

    # Keywords for your package.
    keywords:
      - release
      - makeself

    # Your app's homepage.
    #
    # Default: inferred from global metadata.
    homepage: "https://example.com/"

    # Your software license.
    #
    # Default: inferred from global metadata.
    license: MIT

    # The maintainer/author of the package.
    #
    # Default: inferred from global metadata.
    maintainer: "Foo Bar <foo at bar dot com>"

    # Archive file name template.
    #
    # Mind that the filename must contain the desired extension as well,
    # which typically is `.run`.
    #
    # Default: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}.run'
    # Templates: allowed.
    filename: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.run"

    # Compression format to use.
    #
    # Valid options: gzip, bzip2, xz, lzo, compress, none
    # Default: gzip (Makeself's default)
    # Templates: allowed.
    # Note: none translates to makeself's --nocomp flag
    compression: "gzip"

    # Path to setup script file.
    # This script will be copied into the archive.
    # It is executed when the user runs the Makeself package.
    # Templates: allowed.
    script: install.sh

    # Additional command-line arguments to pass to Makeself.
    #
    # Refer to https://makeself.io for more information.
    #
    # Templates: allowed.
    extra_args:
      - "--notemp"
      - "--needroot"
      - "--keep"
      - "--copy"

    # Additional files/globs you want to add to the Makeself package.
    # These files will be available to the install script.
    #
    # Templates: allowed.
    files:
      - LICENSE.txt
      - README_{{.Os}}.md
      - CHANGELOG.md
      - configs/*
      - scripts/*.sh
      # a more complete example
      - src: "*.md"
        dst: docs

        # Strip parent directories when adding files to the archive.
        strip_parent: true

    # Additional templated files to add to the package.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the source archive.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    templated_files:
      # a more complete example, check the globbing deep dive below
      - src: "LICENSE.md.tpl"
        dst: LICENSE.md

    # Disable this Makeself package.
    # Templates: allowed.
    disable: "{{ .Env.SKIP_MAKESELF }}"
```

Please refer to the [Makeself documentation][Makeself] for more information.

!!! tip

    The install script has access to all files included in the package,
    so you can reference documentation, configuration files,
    or other assets in your installation logic.

!!! tip "Root Privileges"

    When using `--needroot` in `extra_args`, the Makeself installer will
    automatically prompt for root privileges when executed.
    This allows your install script to assume root access without using `sudo`
    commands, making the script simpler and more reliable.

!!! warning

    Makeself packages are platform-specific (typically Linux and macos) and
    create executable files.
    Make sure your target users can execute them on their systems.

<!-- md:templates -->

[Makeself]: https://makeself.io/
