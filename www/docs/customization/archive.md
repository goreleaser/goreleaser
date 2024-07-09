# Archives

The binaries built will be archived together with the `README` and `LICENSE` files into a
`tar.gz` file. In the `archives` section you can customize the archive name,
additional files, and format.

Here is a commented `archives` section with all fields specified:

```yaml
# .goreleaser.yaml
archives:
  - #
    # ID of this archive.
    #
    # Default: 'default'.
    id: my-archive

    # Builds reference which build instances should be archived in this archive.
    builds:
      - default

    # Archive format.
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
    # - `tzst` (since v2.1)
    # - `tar`
    # - `gz`
    # - `zip`
    # - `binary`
    #
    # Default: 'tar.gz'.
    format: zip

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

        # The format to use for the given GOOS.
        #
        # Valid options are `tar.gz`, `tgz`, `tar.xz`, `txz`, tar`, `gz`, `zip`, `binary`, and `none`.
        format: zip

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

{% include-markdown "../includes/pro.md" comments=false %}

{% include-markdown "../includes/templates.md" comments=false %}

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

```yaml
# .goreleaser.yaml
archives:
  - files:
      - none*
```

This would add all files matching the glob `none*`, provide that you don't
have any files matching that glob, only the binary will be added to the
archive.

For more information, check [#602](https://github.com/goreleaser/goreleaser/issues/602)

## A note about Gzip

Gzip is a compression-only format, therefore, it couldn't have more than one
file inside.

Presumably, you'll want that file to be the binary, so, your archive section
will probably look like this:

```yaml
# .goreleaser.yaml
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

## Disable archiving

You can do that by setting `format` to `binary`:

```yaml
# .goreleaser.yaml
archives:
  - format: binary
```

Make sure to check the rest of the documentation above, as doing this has some
implications.

If you have customization that might rely on archives, for instance,
`brews.install`, make sure to fix them too.
