# Makeself Self-Extracting Archives

Makeself creates self-extracting archives that can be executed to automatically
extract and optionally install their contents.
This is particularly useful for distributing software that needs to be easily
installable without requiring users to manually extract archives.
Typically this supports Linux, MacOS and any other platform that makeself runs
on.

!!! note

    The `makeself` command requires the `makeself` command to be available in
    your system `$PATH`.
    You can install it from your system package manager or from
    [the makeself project](https://github.com/megastep/makeself).

Here is a commented `makeselfs` section with all fields specified:

```yaml title=".goreleaser.yaml"
makeselfs:
  - #
    # ID of this makeself package.
    #
    # Default: 'default'.
    id: my-installer

    # IDs of the builds which should be packaged in this makeself archive.
    #
    # Default: empty (include all).
    ids:
      - my-binary

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
    # Default: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}.run'
    # Templates: allowed.
    filename: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.run"

    # Compression format to use.
    #
    # Valid options: gzip, bzip2, xz, lzo, compress, none
    # Default: gzip (makeself default)
    # Templates: allowed.
    # note: none translates to makeslef's --nocomp flag
    compression: "gzip"

    # Path to setup script file.
    # This script will be copied into the archive.
    # It is executed when the user runs the makeself package.
    # Templates: allowed.
    script: install.sh

    # Additional command-line arguments to pass to makeself.
    #
    # Refer to https://makeself.io for more information.
    #
    # Templates: allowed.
    extra_args:
      - "--notemp"
      - "--needroot"
      - "--keep"
      - "--copy"

    # Additional files/globs you want to add to the makeself package.
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

        # File info.
        info:
          # Templates: allowed.
          owner: root

          # Templates: allowed.
          group: root

          # Must be in time.RFC3339Nano format.
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"

          # File mode.
          mode: 0644

    # Disable this makeself package.
    # Templates: allowed.
    disable: "{{ .Env.SKIP_MAKESELF }}"
```

Please refer to the [makeself documentation](https://makeself.io/)
for more information.

!!! tip

    The install script has access to all files included in the package,
    so you can reference documentation, configuration files,
    or other assets in your installation logic.

!!! tip "Root Privileges"

    When using `--needroot` in `extra_args`, the makeself installer will
    automatically prompt for root privileges when executed.
    This allows your install script to assume root access without using `sudo`
    commands, making the script simpler and more reliable.

!!! warning

    Makeself packages are platform-specific (typically Linux and macos) and
    create executable files.
    Make sure your target users can execute them on their systems.

<!-- md:templates -->
