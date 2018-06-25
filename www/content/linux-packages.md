---
title: Linux Packages
series: customization
hideFromIndex: true
weight: 80
---

GoReleaser can generate RPM, Deb and Snap packages for your projects.

Let's see each option in detail:

## NFPM

GoReleaser can be wired to [nfpm](https://github.com/goreleaser/nfpm) to
generate and publish `.deb` and `.rpm` packages.

Available options:

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
  # For RPM packages rpmbuild >= 4.13 is required
  recommends:
    - bzr
    - gtk

  # Packages your package suggests installing.
  # For RPM packages rpmbuild >= 4.13 is required
  suggests:
    - cvs
    - ksh

  # Packages that conflict with your package.
  conflicts:
    - svn
    - bash

  # Override default /usr/local/bin destination for binaries
  bindir: /usr/bin

  # Empty folders that should be created and managed by the packager
  # implementation.
  # Default is empty.
  empty_folders:
  - /var/log/foobar

  # Files or directories to add to your package (beyond the binary).
  # Keys are source paths/globs to get the files from.
  # Values are the destination locations of the files in the package.
  files:
    "scripts/etc/init.d/": "/etc/init.d"
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

Note that GoReleaser will not install `rpmbuild` or any dependencies for you.
As for now, `rpmbuild` is recommended if you want to generate rpm packages.
You can install it with `apt-get install rpm` or `brew install rpm`.

## Snapcraft

GoReleaser can also generate `snap` packages.
[Snaps](http://snapcraft.io/) are a new packaging format, that will let you
publish your project directly to the Ubuntu store.
From there it will be installable in all the
[supported Linux distros](https://snapcraft.io/docs/core/install), with
automatic and transactional updates.

You can read more about it in the [snapcraft docs](https://snapcraft.io/docs/).

Available options:

```yml
# .goreleaser.yml
snapcraft:
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

  # The name of the snap. This is optional.
  # Default is project name.
  name: drumroll

  # Single-line elevator pitch for your amazing snap.
  # 79 char long at most.
  summary: Software to create fast and easy drum rolls.

  # This the description of your snap. You have a paragraph or two to tell the
  # most important story about your snap. Keep it under 100 words though,
  # we live in tweetspace and your description wants to look good in the snap
  # store.
  description: |
    This is the best drum roll application out there.
    Install it and awe!

  # A guardrail to prevent you from releasing a snap to all your users before
  # it is ready.
  # `devel` will let you release only to the `edge` and `beta` channels in the
  # store. `stable` will let you release also to the `candidate` and `stable`
  # channels. More info about channels here:
  # https://snapcraft.io/docs/reference/channels
  grade: stable

  # Snaps can be setup to follow three different confinement policies:
  # `strict`, `devmode` and `classic`. A strict confinement where the snap
  # can only read and write in its own namespace is recommended. Extra
  # permissions for strict snaps can be declared as `plugs` for the app, which
  # are explained later. More info about confinement here:
  # https://snapcraft.io/docs/reference/confinement
  confinement: strict

  # Each binary built by GoReleaser is an app inside the snap. In this section
  # you can declare extra details for those binaries. It is optional.
  apps:

    # The name of the app must be the same name as the binary built.
    drumroll:

      # If your app requires extra permissions to work outside of its default
      # confined space, declare them here.
      # You can read the documentation about the available plugs and the
      # things they allow:
      # https://snapcraft.io/docs/reference/interfaces).
      plugs: ["home", "network"]

      # If you want your app to be autostarted and to always run in the
      # background, you can make it a simple daemon.
      daemon: simple

      # If you any to pass args to your binary, you can add them with the
      # args option.
      args: --foo
```

Note that GoReleaser will not install `snapcraft` nor any of its dependencies
for you.
