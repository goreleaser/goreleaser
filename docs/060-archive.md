---
title: Custom archiving
---

The binaries built will be archived within the README and LICENSE files into a
`tar.gz` file. In the `archive` section you can customize the archive name,
files, and format.

Here is a full commented `archive` section:

```yml
# .goreleaser.yml
archive:
  # You can change the name of the archive.
  # This is parsed with Golang template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Tag with the `v` prefix stripped)
  # - Os
  # - Arch
  # - Arm (ARM version)
  # The default is `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}`
  name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

  # Archive format. Valid options are `tar.gz`, `zip` and `binary`.
  # If format is `binary` no archives are created and the binaries are instead uploaded directly.
  # In that case name_template the below specified files are ignored.
  # Default is `tar.gz`
  format: zip

  # Can be used to archive on different formats for specific GOOSs.
  # Most common use case is to archive as zip on Windows.
  # Default is empty
  format_overrides:
    - goos: windows
      format: zip

  # Replacements for GOOS and GOARCH on the archive name.
  # The keys should be valid GOOS or GOARCH values followed by your custom
  # replacements.
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux

  # Additional files/globs you want to add to the archive.
  # Defaults are any files matching `LICENCE*`, `LICENSE*`,
  # `README*` and `CHANGELOG*` (case-insensitive)
  files:
    - LICENSE.txt
    - README.md
    - CHANGELOG.md
    - docs/*
    - design/*.png
```
