---
title: NFPM
---

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb` and `.rpm` packages.

Available options:

```yaml
# .goreleaser.yml
nfpms:
  # note that this is an array of nfpm configs
  -
    # ID of the nfpm config, must be unique.
    # Defaults to "default".
    id: foo

    # Name of the package.
    # Defaults to `ProjectName`.
    package_name: foo

    # You can change the file name of the package.
    # Default: `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}`
    file_name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # Build IDs for the builds you want to create NFPM packages for.
    # Defaults to all builds.
    builds:
      - foo
      - bar

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
      - apk
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

    # Package epoch.
    # Defaults to empty.
    epoch: 1

    # Package release.
    # Defaults to empty.
    release: 1

    # Makes a meta package - an empty package that contains only supporting files and dependencies.
    # When set to `true`, the `builds` option is ignored.
    # Defaults to false.
    meta: true

    # Empty folders that should be created and managed by the packager
    # implementation.
    # Default is empty.
    empty_folders:
      - /var/log/foobar

    # Files to add to your package (beyond the binary).
    # Keys are source paths/globs to get the files from.
    # Values are the destination locations of the files in the package.
    # Use globs to add all contents of a folder.
    files:
      "scripts/etc/init.d/**": "/etc/init.d"
      "path/**/glob": "/var/foo/glob"

    # Config files to add to your package. They are about the same as
    # the files keyword, except package managers treat them differently (while
    # uninstalling, mostly).
    # Keys are source paths/globs to get the files from.
    # Values are the destination locations of the files in the package.
    config_files:
      "tmp/app_generated.conf": "/etc/app.conf"
      "conf/*.conf": "/etc/foo/"

    # Scripts to execute during the installation of the package.
    # Keys are the possible targets during the installation process
    # Values are the paths to the scripts which will be executed
    scripts:
      preinstall: "scripts/preinstall.sh"
      postinstall: "scripts/postinstall.sh"
      preremove: "scripts/preremove.sh"
      postremove: "scripts/postremove.sh"

    # Some attributes can be overrided per package format.
    overrides:
      deb:
        conflicts:
          - subversion
        dependencies:
          - git
        suggests:
          - gitk
        recommends:
          - tig
        empty_folders:
          - /var/log/bar
      rpm:
        replacements:
          amd64: x86_64
        name_template: "{{ .ProjectName }}-{{ .Version }}-{{ .Arch }}"
        files:
          "tmp/man.gz": "/usr/share/man/man8/app.8.gz"
        config_files:
          "tmp/app_generated.conf": "/etc/app-rpm.conf"
        scripts:
          preinstall: "scripts/preinstall-rpm.sh"
```

!!! tip
    Learn more about the [name template engine](/customization/templates).
