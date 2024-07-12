# Snapcraft Packages (snaps)

GoReleaser can also generate `snap` packages.
[Snaps](http://snapcraft.io/) are a new packaging format, that will let you
publish your project directly to the Ubuntu store.
From there it will be installable in all the
[supported Linux distros](https://snapcraft.io/docs/core/install), with
automatic and transactional updates.

!!! warning

    Snapcraft packages can't be build inside a Docker container.

You can read more about it in the [snapcraft docs](https://snapcraft.io/docs/).

Available options:

```yaml
# .goreleaser.yaml
snapcrafts:
  - #
    # ID of the snapcraft config, must be unique.
    #
    # Default: 'default'.
    id: foo

    # Build IDs for the builds you want to create snapcraft packages for.
    builds:
      - foo
      - bar

    # You can change the name of the package.
    #
    # Default: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'.
    # Templates: allowed.
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # The name of the snap. This is optional.
    #
    # Default: the project name.
    name: drumroll

    # The canonical title of the application, displayed in the software
    # centre graphical frontends.
    title: Drum Roll

    # Path to icon image that represents the snap in the snapcraft.io store
    # pages and other graphical store fronts.
    icon: ./icon.png

    # Whether to publish the snap to the snapcraft store.
    # Remember you need to `snapcraft login` first.
    publish: true

    # Single-line elevator pitch for your amazing snap.
    # 79 char long at most.
    summary: Software to create fast and easy drum rolls.

    # This the description of your snap. You have a paragraph or two to tell the
    # most important story about your snap. Keep it under 100 words though,
    # we live in tweetspace and your description wants to look good in the snap
    # store.
    description: This is the best drum roll application out there. Install it and awe!

    # Disable this configuration.
    #
    # Templates: allowed.
    disable: true

    # Channels in store where snap will be pushed.
    #
    # More info about channels here:
    # https://snapcraft.io/docs/reference/channels
    #
    # Default:
    #   grade is 'stable': ["edge", "beta", "candidate", "stable"]
    #   grade is 'devel': ["edge", "beta"]
    # Templates: allowed.
    channel_templates:
      - edge
      - beta
      - candidate
      - stable
      - "{{ .Major }}.{{ .Minor }}/edge"
      - "{{ .Major }}.{{ .Minor }}/beta"
      - "{{ .Major }}.{{ .Minor }}/candidate"
      - "{{ .Major }}.{{ .Minor }}/stable"

    # A guardrail to prevent you from releasing a snap to all your users before
    # it is ready.
    # `devel` will let you release only to the `edge` and `beta` channels in the
    # store. `stable` will let you release also to the `candidate` and `stable`
    # channels.
    #
    # Default: 'stable'
    grade: stable

    # Snaps can be setup to follow three different confinement policies:
    # `strict`, `devmode` and `classic`. A strict confinement where the snap
    # can only read and write in its own namespace is recommended. Extra
    # permissions for strict snaps can be declared as `plugs` for the app, which
    # are explained later. More info about confinement here:
    # https://snapcraft.io/docs/reference/confinement
    #
    # Default: 'strict'
    confinement: strict

    # Your app's license, based on SPDX license expressions:
    # https://spdx.org/licenses
    license: MIT

    # A snap of type base to be used as the execution environment for this snap.
    # Valid values are:
    # * bare - Empty base snap;
    # * core - Ubuntu Core 16;
    # * core18 - Ubuntu Core 18.
    base: core18

    # A list of features that must be supported by the core in order for
    # this snap to install.
    assumes:
      - snapd2.38

    # his top-level keyword to define a hook with a plug to access more
    # privileges.
    hooks:
      install:
        - network

    # Add extra files on the resulting snap. Useful for including wrapper
    # scripts or other useful static files. Source filenames are relative to the
    # project directory. Destination filenames are relative to the snap prime
    # directory.
    extra_files:
      - source: drumroll.wrapper
        destination: bin/drumroll.wrapper
        mode: 0755

    # Additional templated extra files to add to the package.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the package.
    #
    # This feature is only available in GoReleaser Pro.
    # Templates: allowed.
    templated_extra_files:
      - source: LICENSE.tpl
        destination: LICENSE.txt
        mode: 0644

    # With layouts, you can make elements in $SNAP, $SNAP_DATA, $SNAP_COMMON
    # accessible from locations such as /usr, /var and /etc. This helps when
    # using pre-compiled binaries and libraries that expect to find files and
    # directories outside of locations referenced by $SNAP or $SNAP_DATA.
    # About snap environment variables:
    # * HOME: set to SNAP_USER_DATA for all commands
    # * SNAP: read-only install directory
    # * SNAP_ARCH: the architecture of device (eg, amd64, arm64, armhf, i386, etc)
    # * SNAP_DATA: writable area for a particular revision of the snap
    # * SNAP_COMMON: writable area common across all revisions of the snap
    # * SNAP_LIBRARY_PATH: additional directories which should be added to LD_LIBRARY_PATH
    # * SNAP_NAME: snap name
    # * SNAP_INSTANCE_NAME: snap instance name incl. instance key if one is set (snapd 2.36+)
    # * SNAP_INSTANCE_KEY: instance key if any (snapd 2.36+)
    # * SNAP_REVISION: store revision of the snap
    # * SNAP_USER_DATA: per-user writable area for a particular revision of the snap
    # * SNAP_USER_COMMON: per-user writable area common across all revisions of the snap
    # * SNAP_VERSION: snap version (from snap.yaml)
    # More info about layout here:
    # https://snapcraft.io/docs/snap-layouts
    layout:
      # The path you want to access in sandbox.
      /etc/drumroll:
        # Which outside file or directory you want to map to sandbox.
        # Valid keys are:
        # * bind - Bind-mount a directory.
        # * bind_file - Bind-mount a file.
        # * symlink - Create a symbolic link.
        # * type - Mount a private temporary in-memory filesystem.
        bind: $SNAP_DATA/etc

    # Each binary built by GoReleaser is an app inside the snap. In this section
    # you can declare extra details for those binaries. It is optional.
    # See: https://snapcraft.io/docs/snapcraft-app-and-service-metadata
    apps:
      # The name of the app must be the same name as the binary built or the snapcraft name.
      drumroll:
        # If you any to pass args to your binary, you can add them with the
        # args option.
        args: --foo

        # The kind of wrapper to generate for the given command.
        adapter: none

        # List of applications that are ordered to be started after the current
        # one.
        after: ["postdrum"]

        # Aliases for the app command.
        # https://snapcraft.io/docs/commands-and-aliases#heading--aliases
        aliases: ["droll"]

        # Defines the name of the .desktop file used to start an application
        # with the desktop session.
        # https://snapcraft.io/docs/snap-format#heading--autostart
        autostart: drumroll.desktop

        # List of applications that are ordered to be started before the current
        # one.
        before: ["predrum"]

        # D-Bus name this service is reachable as. Mandatory if daemon=dbus.
        bus_name: drumbus

        # A list of commands to be executed in order before the command of this
        # app.
        command_chain: ["foo", "bar", "baz"]

        # An identifier to a desktop-id within an external appstream file.
        # https://snapcraft.io/docs/using-external-metadata
        common_id: "com.example.drumroll"

        # Bash completion snippet. More information about completion here:
        # https://snapcraft.io/docs/tab-completion.
        completer: drumroll-completion.bash

        # You can override the command name.
        #
        # Default: AppName.
        command: bin/drumroll.wrapper

        # If you want your app to be autostarted and to always run in the
        # background, you can make it a simple daemon.
        daemon: simple

        # Location of the .desktop file.
        desktop: usr/share/applications/drumroll.desktop

        # A set of key-value pairs specifying environment variables.
        environment:
          foo: bar
          baz: quo

        # A list of Snapcraft extensions this app depends on.
        # https://snapcraft.io/docs/snapcraft-extensions
        extensions: ["gnome-3-38"]

        # Defines whether a freshly installed daemon is started automatically,
        # or whether startup control is deferred to the snap.
        # Requires `daemon` to be set.
        install_mode: "disable"

        # A set of key-value attributes passed through to snap.yaml without
        # snapcraft validation.
        # https://snapcraft.io/docs/using-in-development-features
        passthrough:
          foo: bar

        # If your app requires extra permissions to work outside of its default
        # confined space, declare them here.
        # You can read the documentation about the available plugs and the
        # things they allow:
        # https://snapcraft.io/docs/supported-interfaces.
        plugs: ["home", "network", "personal-files"]

        # Sets a command to run from inside the snap after a service stops.
        post_stop_command: foo

        # Controls whether the daemon should be restarted during a snap refresh.
        refresh_mode: endure

        # Command to use to ask the service to reload its configuration.
        # Requires `daemon` to be set.
        reload_command: foo

        # Restart condition of the snap.
        # https://snapcraft.io/docs/snapcraft-yaml-reference
        restart_condition: "always"

        # List of slots for interfaces to connect to.
        slots: ["foo", "bar", "baz"]

        # Maps a daemon’s sockets to services and activates them.
        # Requires `plugs` to contain `network-bind`.
        sockets:
          sock:
            listen-stream: $SNAP_COMMON/socket
            socket-group: socket-group
            socket-mode: 416

        # Time to wait for daemon to start.
        start_timeout: 42ms

        # Command to use to stop the service.
        # Requires `daemon` to be set.
        stop_command: foo

        # Controls how the daemon should be stopped.
        # Requires `daemon` to be set.
        stop_mode: sigterm

        # Time to wait for daemon to stop.
        stop_timeout: 42ms

        # Schedules when, or how often, to run a service or command.
        # Requires `daemon` to be set.
        # https://snapcraft.io/docs/services-and-daemons
        timer: "00:00-24:00/24"

        # Declares the service watchdog timeout.
        # Requires `plugs` to contain `daemon-notify`.
        watchdog_timeout: 42ms

    # Allows plugs to be configured. Plugs like system-files and personal-files
    # require this.
    plugs:
      personal-files:
        read:
          - $HOME/.foo
        write:
          - $HOME/.foo
          - $HOME/.foobar
```

{% include-markdown "../includes/templates.md" comments=false %}

!!! note

    GoReleaser will not install `snapcraft` nor any of its dependencies for you.
