---
title: "Flatpak Packages"
linkTitle: Flatpak
weight: 110
---

{{< version "v2.15" >}}

GoReleaser can create [Flatpak][] bundles (`.flatpak` files) for your Linux
binaries.
Flatpak is a framework for distributing desktop applications across various
Linux distributions, providing sandboxed environments with consistent runtimes.

The resulting `.flatpak` bundles can be uploaded to releases, blob
storage, or distributed directly to users.

> [!NOTE]
> This feature requires `flatpak-builder` and `flatpak` to be available in
> your system `$PATH`.
> You can install them from your system package manager.
> The configured runtime and SDK must also be installed on the build machine.
> Flatpak only works from Linux.

## Configuration

Here is a commented `flatpaks` section with all fields specified:

```yaml {filename=".goreleaser.yaml"}
flatpak:
  - #
    # ID of this Flatpak config, must be unique.
    #
    # Default: 'default'.
    id: foo

    # IDs of the builds which should be packaged.
    #
    # Default: empty (include all).
    ids:
      - foo
      - bar

    # Flatpak bundle file name template.
    #
    # Default: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}'.
    # Templates: allowed.
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

    # The Flatpak application ID.
    # Must follow the reverse-DNS naming convention (e.g. com.example.App).
    #
    # Required.
    app_id: com.example.MyApp

    # The Flatpak runtime to use.
    #
    # Required.
    runtime: org.freedesktop.Platform

    # The Flatpak runtime version.
    #
    # Required.
    runtime_version: "24.08"

    # The Flatpak SDK to use.
    # Must be compatible with the chosen runtime.
    #
    # Required.
    sdk: org.freedesktop.Sdk

    # The command to run inside the Flatpak.
    #
    # Default: the first binary name.
    command: my-app

    # Permissions to grant to the sandboxed application.
    # See https://docs.flatpak.org/en/latest/sandbox-permissions.html
    #
    # Default: empty.
    finish_args:
      - --share=network
      - --share=ipc
      - --socket=x11
      - --socket=wayland
      - --filesystem=home

    # Disable this Flatpak package.
    #
    # Templates: allowed.
    disable: "{{ .Env.SKIP_FLATPAK }}"
```

{{< templates >}}

> [!NOTE]
> **Supported Architectures**
>
> GoReleaser maps Go architectures to Flatpak architectures automatically:
>
> | Go arch | Flatpak arch |
> | ------- | ------------ |
> | `amd64` | `x86_64`     |
> | `arm64` | `aarch64`    |
>
> Other architectures are ignored.

> [!WARNING]
> Flatpak packages require the configured runtime (e.g.
> `org.freedesktop.Platform`) and SDK to be installed on the build machine.
> Make sure to install them before running GoReleaser:
>
> ```bash
> flatpak install flathub \
>   org.freedesktop.Platform/x86_64/24.08 \
>   org.freedesktop.Sdk/x86_64/24.08 \
>   org.freedesktop.Platform/aarch64/24.08 \
>   org.freedesktop.Sdk/aarch64/24.08
>
> ```

[Flatpak]: https://flatpak.org/
