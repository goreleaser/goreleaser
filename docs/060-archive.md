---
title: Archive
---

The binaries built will be archived together with the `README` and `LICENSE` files into a
`tar.gz` file. In the `archive` section you can customize the archive name,
additional files, and format.

Here is a commented `archive` section with all fields specified:

```yml
# .goreleaser.yml
archive:
  # You can change the name of the archive.
  # This is parsed with the Go template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Git tag without `v` prefix)
  # - Os
  # - Arch
  # - Arm (ARM version)
  # Default is `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}`.
  name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

  # Set to true, if you want all files in the archive to be in a single directory.
  # If set to true and you extract the archive 'goreleaser_Linux_arm64.tar.gz',
  # you get a folder 'goreleaser_Linux_arm64'.
  # If set to false, all files are extracted separately.
  # Default is false.
  wrap_in_directory: true

  # Archive format. Valid options are `tar.gz`, `zip` and `binary`.
  # If format is `binary`, no archives are created and the binaries are instead uploaded directly.
  # In that case name_template and the below specified files are ignored.
  # Default is `tar.gz`.
  format: zip

  # Can be used to change the archive formats for specific GOOSs.
  # Most common use case is to archive as zip on Windows.
  # Default is empty.
  format_overrides:
    - goos: windows
      format: zip

  # Replacements for GOOS and GOARCH in the archive name.
  # Keys should be valid GOOSs or GOARCHs.
  # Values are the respective replacements.
  # Default is empty.
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux

  # Additional files/globs you want to add to the archive.
  # Defaults are any files matching `LICENCE*`, `LICENSE*`,
  # `README*` and `CHANGELOG*` (case-insensitive).
  files:
    - LICENSE.txt
    - README.md
    - CHANGELOG.md
    - docs/*
    - design/*.png
```
