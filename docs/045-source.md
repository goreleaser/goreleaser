---
title: Source
---

The source code used to build a relase is captured in an archive which can later
be used to support homebrew formulae building from source.

Here is a commented `source` section with all fields specified:

```yml
# .goreleaser.yml
source:
  # This is parsed with the Go template engine and the following variables
  # are available:
  # - Binary
  # - Tag
  # - Version
  name_template: "{{.Binary}}-{{.Version}}"
  # Excludes are matched against the filename with filepath.Match and if they
  # match the file won't be included in the archive.
  # Default is empty.
  excludes:
    - "dist/*"
```
