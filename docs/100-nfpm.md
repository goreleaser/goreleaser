---
title: NFPM
---

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb` and `.rpm` packages.

```yml
# .goreleaser.yml
nfpm:
  # You can change the name of the package.
  # This is parsed with the Go template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Git tag without `v` prefix)
  # - Os
  # - Arch
  # - Arm (ARM version)
  # - Env (environment variables)
  # Default: `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}`
  name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

  # Replacements for GOOS and GOARCH in the package name.
  # Keys should be valid GOOSs or GOARCHs.
  # Values are the respective replacements.
  # Default is empty.
  replacements:
    amd64: 64-bit
    386: 32-bit
    darwin: macOS
    linux: Tux

  # Your app's vendor.
  # Default is empty.
  vendor: Drum Roll Inc.
  # Your app's homepage.
  # Default is empty.
  homepage: https://example.com/

  # Your app's maintainer (probably you).
  # Default is empty.
  maintainer: Drummer <drum-roll@example.com>

  # Your app's description.
  # Default is empty.
  description: Software to create fast and easy drum rolls.

  # Your app's license.
  # Default is empty.
  license: Apache 2.0

  # Formats to be generated.
  formats:
    - deb
    - rpm

  # Packages your package depends on.
  dependencies:
    - git
    - zsh

  # Packages your package recommends installing.
  recommends:
    - bzr
    - gtk

  # Packages your package suggests installing.
  suggests:
    - cvs
    - ksh

  # Packages that conflict with your package.
  conflicts:
    - svn
    - bash

  # Override default /usr/local/bin destination for binaries
  bindir: /usr/bin

  # Files or directories to add to your package (beyond the binary).
  # Keys are source paths to get the files from.
  # Values are the destination locations of the files in the package.
  files:
    "scripts/etc/init.d/": "/etc/init.d"
```

Note that GoReleaser will not install `rpmbuild` or any dependencies for you.
As for now, `rpmbuild` is recommended if you want to generate rpm packages.
You can install it with `apt-get install rpm` or `brew install rpm`.
