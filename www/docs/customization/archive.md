# Archives

The binaries built will be archived together with the `README` and `LICENSE` files into a
`tar.gz` file. In the `archives` section you can customize the archive name,
additional files, and format.

Here is a commented `archives` section with all fields specified:

```yaml title=".goreleaser.yaml"
archives:
  - #
    # ID of this archive.
    #
    # Default: 'default'.
    id: my-archive

    # IDs of the builds which should be archived in this archive.
    #
    # <!-- md:inline_version v2.8 --> (use 'builds' in previous versions).
    # Default: empty (include all).
    ids:
      - default

    # Archive formats.
    #
    # If format is `binary`, no archives are created and the binaries are instead
    # uploaded directly.
    #
    # Valid options are:
    # - `tar.gz`
    # - `tgz`
    # - `tar.xz`
    # - `txz`
    # - `tar.zst`
    # - `tzst` # <!-- md:inline_version v2.1 -->.
    # - `tar`
    # - `gz`
    # - `zip`
    # - `makeself` # <!-- md:inline_version v2.12 -->.
    # - `binary`
    #
    # Default: ['tar.gz'].
    format: "zip" # Singular form, single format, deprecated.
    formats: ["zip", "tar.gz"] # Plural form, multiple formats. <!-- md:inline_version v2.6 -->

    # This will create an archive without any binaries, only the files are there.
    # The name template must not contain any references to `Os`, `Arch` and etc, since the archive will be meta.
    #
    # Templates: allowed.
    meta: true

    # Archive name.
    #
    # Default:
    # - if format is `binary`:
    #   - `{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
    # - if format is anything else:
    #   - `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`
    # Templates: allowed.
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # Sets the given file info to all the binaries included from the `builds`.
    #
    # Default: copied from the source binary.
    builds_info:
      group: root
      owner: root
      mode: 0644
      # format is `time.RFC3339Nano`
      mtime: 2008-01-02T15:04:05Z

    # Set this to true if you want all files in the archive to be in a single directory.
    # If set to true and you extract the archive 'goreleaser_Linux_arm64.tar.gz',
    # you'll get a directory 'goreleaser_Linux_arm64'.
    # If set to false, all files are extracted separately.
    # You can also set it to a custom directory name (templating is supported).
    wrap_in_directory: true

    # If set to true, will strip the parent directories away from binary files.
    #
    # This might be useful if you have your binary be built with a sub-directory
    # for some reason, but do no want that sub-directory inside the archive.
    strip_binary_directory: true

    # This will make the destination paths be relative to the longest common
    # path prefix between all the files matched and the source glob.
    # Enabling this essentially mimic the behavior of nfpm's contents section.
    # It will be the default by June 2023.
    rlcp: true

    # Can be used to change the archive formats for specific GOOSs.
    # Most common use case is to archive as zip on Windows.
    format_overrides:
      - # Which GOOS to override the format for.
        goos: windows

        # The formats to use for the given GOOS.
        #
        # Valid options are:
        # - `tar.gz`
        # - `tgz`
        # - `tar.xz`
        # - `txz`
        # - `tar.zst`
        # - `tzst` # <!-- md:inline_version v2.1 -->.
        # - `tar`
        # - `gz`
        # - `zip`
        # - `makeself` # <!-- md:inline_version v2.12 -->.
        # - `binary` # be extra-cautious with the file name template in this case!
        # - `none`   # skips this archive
        #
        format: "zip" # Singular form, single format, deprecated.
        formats: ["zip", "tar.gz"] # Plural form, multiple formats. <!-- md:inline_version v2.6 -->

    # Additional files/globs you want to add to the archive.
    #
    # Default: [ 'LICENSE*', 'README*', 'CHANGELOG', 'license*', 'readme*', 'changelog'].
    # Templates: allowed.
    files:
      - LICENSE.txt
      - README_{{.Os}}.md
      - CHANGELOG.md
      - docs/*
      - design/*.png
      - templates/**/*
      # a more complete example, check the globbing deep dive below
      - src: "*.md"
        dst: docs

        # Strip parent directories when adding files to the archive.
        strip_parent: true

        # File info.
        # Not all fields are supported by all formats available formats.
        #
        # Default: copied from the source file.
        info:
          # Templates: allowed.
          owner: root

          # Templates: allowed.
          group: root

          # Must be in time.RFC3339Nano format.
          #
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"

          # File mode.
          mode: 0644

    # Additional templated files to add to the archive.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the archive.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    templated_files:
      # a more complete example, check the globbing deep dive below
      - src: "LICENSE.md.tpl"
        dst: LICENSE.md

        # File info.
        # Not all fields are supported by all formats available formats.
        #
        # Default: copied from the source file.
        info:
          # Templates: allowed.
          owner: root

          # Templates: allowed.
          group: root

          # Must be in time.RFC3339Nano format.
          #
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"

          # File mode.
          mode: 0644

    # Before and after hooks for each archive.
    # Skipped if archive format is binary.
    # If multiple formats are set, hooks will be executed for each format.
    # Extra template fields available: `.Format`.
    # This feature is only available in GoReleaser Pro.
    hooks:
      before:
        - make clean # simple string
        - cmd: go generate ./... # specify cmd
        - cmd: go mod tidy
          output: true # always prints command output
          dir: ./submodule # specify command working directory
        - cmd: touch {{ .Env.FILE_TO_TOUCH }}
          env:
            - "FILE_TO_TOUCH=something-{{ .ProjectName }}" # specify hook level environment variables

      after:
        - make clean
        - cmd: cat *.yaml
          dir: ./submodule
        - cmd: touch {{ .Env.RELEASE_DONE }}
          env:
            - "RELEASE_DONE=something-{{ .ProjectName }}" # specify hook level environment variables

    # Disables the binary count check.
    allow_different_binary_count: true
```

<!-- md:pro -->

<!-- md:templates -->

!!! tip

    You can add entire directories, its sub-directories and files by using the
    glob notation, for example: `mydirectory/**/*`.

!!! warning

    The `files` and `wrap_in_directory` options are ignored if `format` is `binary`.

!!! warning

    The `name_template` option will not reflect the filenames under the `dist`
    directory if `format` is `binary`.
    The template will be applied only where the binaries are uploaded (e.g.
    GitHub releases).

## Deep diving into the globbing options

We'll walk through what happens in each case using some examples.

```yaml
# ...
files:
  # Adds `README.md` at the root of the archive:
  - README.md

  # Adds all `md` files to the root of the archive:
  - "*.md"

  # Adds all `md` files to the root of the archive:
  - src: "*.md"

  # Adds all `md` files in the current directory to a `docs` directory in the
  # archive:
  - src: "*.md"
    dst: docs

  # Recursively adds all `go` files to a `source` directory in the archive.
  # in this case, `cmd/myapp/main.go` will be added as `source/cmd/myapp/main.go`
  - src: "**/*.go"
    dst: source

  # Recursively adds all `go` files to a `source` directory in the archive,
  # stripping their parent directory.
  # In this case, `cmd/myapp/main.go` will be added as `source/main.go`:
  - src: "**/*.go"
    dst: source
    strip_parent: true
# ...
```

## Packaging only the binaries

Since GoReleaser will always add the `README` and `LICENSE` files to the
archive if the file list is empty, you'll need to provide a filled `files`
on the archive section.

A working hack is to use something like this:

```yaml title=".goreleaser.yaml"
archives:
  - files:
      - none*
```

This would add all files matching the glob `none*`, provide that you don't
have any files matching that glob, only the binary will be added to the
archive. Any glob that doesn't match any file should work.

For more information, check [#602](https://github.com/goreleaser/goreleaser/issues/602)

## A note about Gzip

Gzip is a compression-only format, therefore, it couldn't have more than one
file inside.

Presumably, you'll want that file to be the binary, so, your archive section
will probably look like this:

```yaml title=".goreleaser.yaml"
archives:
  - format: gz
    files:
      - none*
```

This should create `.gz` files with the binaries only, which should be
extracted with something like `gzip -d file.gz`.

!!! warning

    You won't be able to package multiple builds in a single archive either.
    The alternative is to declare multiple archives filtering by build ID.

## Do not archive

If you want to publish the binaries directly, without any archiving, you can do
so by setting `format` to `binary`:

```yaml title=".goreleaser.yaml"
archives:
  - format: binary
```

You can then set a custom `name_template`, which will be the name used when
uploading the binary to the release, for example.

## Makeself Self-Extracting Archives

Makeself creates self-extracting archives that can be executed to automatically extract and optionally install their contents. This is particularly useful for distributing software that needs to be easily installable without requiring users to manually extract archives.

!!! note

    The `makeself` format requires the `makeself` or `makeself.sh` command to be available in your system PATH. You can install it from your system package manager or from [the makeself project](https://github.com/megastep/makeself).

### Makeself Configuration

When using the `makeself` format, you can configure additional options:

```yaml title=".goreleaser.yaml"
archives:
  - formats: ["makeself"]
    
    # Makeself-specific configuration
    makeself:
      # Custom file extension for the self-extracting archive.
      # Default: '.run'
      # Templates: allowed.
      extension: ".run"
      
      # Custom label/description for the archive.
      # Default: 'Self-extracting archive'
      # Templates: allowed.
      label: "{{ .ProjectName }} v{{ .Version }} Installer"
      
      # Compression format to use.
      # Valid options: gzip, bzip2, xz, lzo, compress, none
      # Default: gzip (makeself default)
      # Templates: allowed.
      compression: "gzip"
      
      # Inline install script content.
      # This script will be executed after extraction.
      # Templates: allowed.
      install_script: |
        #!/bin/bash
        echo "Installing {{ .ProjectName }}..."
        chmod +x {{ .Binary }}
        cp {{ .Binary }} /usr/local/bin/
        echo "Installation complete!"
      
      # Path to install script file within the archive.
      # Alternative to install_script for external script files.
      # Templates: allowed.
      install_script_file: "install.sh"
      
      # Additional command-line arguments to pass to makeself.
      # Templates: allowed.
      extra_args:
        - "--notemp"
        - "--needroot"  # Requires root privileges to run
      
      # Linux Software Map (LSM) template content.
      # Templates: allowed.
      lsm_template: |
        Begin4
        Title: {{ .ProjectName }}
        Version: {{ .Version }}
        Description: {{ .ProjectName }} self-extracting installer
        Author: Your Name
        Maintained-by: your-email@example.com
        Primary-site: https://github.com/youruser/yourproject
        Platforms: Linux
        Copying-policy: MIT
        End
      
      # Path to external LSM file.
      # Alternative to lsm_template for external LSM files.
      # Templates: allowed.
      lsm_file: "project.lsm"
```

### Simple Makeself Example

Here's a minimal example to create a self-extracting installer:

```yaml title=".goreleaser.yaml"
archives:
  - id: installer
    formats: ["makeself"]
    makeself:
      label: "{{ .ProjectName }} v{{ .Version }} Installer"
      extra_args:
        - "--needroot"  # Ensures installer runs as root
      install_script: |
        #!/bin/bash
        echo "Installing {{ .ProjectName }}..."
        chmod +x {{ .Binary }}
        cp {{ .Binary }} /usr/local/bin/
        echo "{{ .ProjectName }} installed successfully!"
```

This will create a `.run` file that users can execute with `./yourproject_1.0.0_linux_amd64.run`. The `--needroot` flag ensures the installer automatically requests root privileges, so the install script can assume it's running as root (no `sudo` needed in the script).

### Advanced Makeself Features

- **Custom Extensions**: Use templated extensions like `.{{ .Os }}.run` for platform-specific naming
- **LSM Support**: Include Linux Software Map information for software catalogs
- **Flexible Install Scripts**: Use either inline scripts or external script files from your repository
- **Compression Options**: Choose from multiple compression formats based on size vs. speed trade-offs
- **Integration with Files**: Add documentation, licenses, and other files that will be available to the install script
- **Root Privileges**: Use `--needroot` in `extra_args` to ensure the installer runs with root privileges, simplifying system-wide installations

!!! tip

    The install script has access to all files included in the archive, so you can reference documentation, configuration files, or other assets in your installation logic.

!!! tip "Root Privileges"

    When using `--needroot` in `extra_args`, the makeself installer will automatically prompt for root privileges when executed. This allows your install script to assume root access without using `sudo` commands, making the script simpler and more reliable. Users will be prompted like: `"This installer requires root privileges. Please enter your password when prompted."``
