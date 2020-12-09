---
title: Snapcraft
---

GoReleaser can also generate `snap` packages.
[Snaps](http://snapcraft.io/) are a new packaging format, that will let you
publish your project directly to the Ubuntu store.
From there it will be installable in all the
[supported Linux distros](https://snapcraft.io/docs/core/install), with
automatic and transactional updates.

You can read more about it in the [snapcraft docs](https://snapcraft.io/docs/).

Available options:

```yaml
# .goreleaser.yml
snapcrafts:
  -
    # ID of the snapcraft config, must be unique.
    # Defaults to "default".
    id: foo

    # Build IDs for the builds you want to create snapcraft packages for.
    # Defaults to all builds.
    builds:
    - foo
    - bar

    # You can change the name of the package.
    # Default: `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}{{ if .Mips }}_{{ .Mips }}{{ end }}`
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

    # Wether to publish the snap to the snapcraft store.
    # Remember you need to `snapcraft login` first.
    # Defaults to false.
    publish: true

    # Single-line elevator pitch for your amazing snap.
    # 79 char long at most.
    summary: Software to create fast and easy drum rolls.

    # This the description of your snap. You have a paragraph or two to tell the
    # most important story about your snap. Keep it under 100 words though,
    # we live in tweetspace and your description wants to look good in the snap
    # store.
    description: This is the best drum roll application out there. Install it and awe!

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

    # Your app's license, based on SPDX license expressions: https://spdx.org/licenses
    # Default is empty.
    license: MIT

    # A snap of type base to be used as the execution environment for this snap.
    # Valid values are:
    # * bare - Empty base snap;
    # * core - Ubuntu Core 16;
    # * core18 - Ubuntu Core 18.
    # Default is empty.
    base: core18

    # Add extra files on the resulting snap. Useful for including wrapper
    # scripts or other useful static files. Source filenames are relative to the
    # project directory. Destination filenames are relative to the snap prime
    # directory.
    # Default is empty.
    extra_files:
      - source: drumroll.wrapper
        destination: bin/drumroll.wrapper
        mode: 0755

    # Each binary built by GoReleaser is an app inside the snap. In this section
    # you can declare extra details for those binaries. It is optional.
    apps:

      # The name of the app must be the same name as the binary built or the snapcraft name.
      drumroll:

        # If your app requires extra permissions to work outside of its default
        # confined space, declare them here.
        # You can read the documentation about the available plugs and the
        # things they allow:
        # https://snapcraft.io/docs/reference/interfaces.
        plugs: ["home", "network", "personal-files"]

        # If you want your app to be autostarted and to always run in the
        # background, you can make it a simple daemon.
        daemon: simple

        # If you any to pass args to your binary, you can add them with the
        # args option.
        args: --foo

        # Bash completion snippet. More information about completion here:
        # https://docs.snapcraft.io/tab-completion-for-snaps.
        completer: drumroll-completion.bash

        # You can override the command name.
        # Defaults is the app name.
        command: bin/drumroll.wrapper

        # Restart condition of the snap.
        # Defaults to empty.
        # https://snapcraft.io/docs/snapcraft-yaml-reference
        restart_condition: "always"

    # Allows plugs to be configured. Plugs like system-files and personal-files
    # require this.
    # Default is empty.
    plugs:
      personal-files:
        read:
        - $HOME/.foo
        write:
        - $HOME/.foo
        - $HOME/.foobar
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

!!! note
    GoReleaser will not install `snapcraft` nor any of its dependencies for you.
