---
title: Announcing GoReleaser v2.15
date: 2026-03-29
slug: goreleaser-v2.15
tags: [announcements]
authors: [caarlos0]
---

This version a big one for Linux packaging - Flatpak bundles and Source RPMs land in
the same release, alongside a rebuilt documentation website and better Go build
defaults.

<!--more-->

## Flatpak

GoReleaser can now build and publish [Flatpak](https://flatpak.org/) bundles!

Flatpak lets you distribute Linux desktop applications as self-contained
bundles that run on any distro, inside a sandboxed environment with a
predictable runtime. A minimal example:

```yaml {filename=".goreleaser.yaml"}
flatpak:
  - app_id: com.example.MyApp
    runtime: org.freedesktop.Platform
    runtime_version: "24.08"
    sdk: org.freedesktop.Sdk
```

The `app_id`, `runtime`, `runtime_version`, and `sdk` fields are required.
You can also set `finish_args` to configure sandbox permissions:

```yaml {filename=".goreleaser.yaml"}
flatpak:
  - app_id: com.example.MyApp
    runtime: org.freedesktop.Platform
    runtime_version: "24.08"
    sdk: org.freedesktop.Sdk
    finish_args:
      - --share=network
      - --socket=wayland
      - --filesystem=home
```

Flatpak bundles are automatically included in checksums and signing.

See the [documentation](https://goreleaser.com/customization/flatpak/) for more
details.

## Go build `./...` and better defaults

The Go builder now supports `./...` as a build target.
GoReleaser will find all packages with a `main` function, infer their binary
names just like `go build` does, and build them all at once:

```yaml {filename=".goreleaser.yaml"}
builds:
  - main: ./...
```

Less config, and faster builds — Go can schedule all packages together in a
single compiler invocation, which matters for repos with multiple binaries.

`goreleaser init` also generates better defaults now, including `goarch` and
`main`, so there's even less to fill in by hand.

## Generate completions in Homebrew Casks

Homebrew Casks now support the `generate_completions_from_executable` stanza.
Users installing your tool via Homebrew will get shell completions
automatically, without any extra manual steps:

```yaml {filename=".goreleaser.yaml"}
homebrew_casks:
  - generate_completions_from_executable: true
```

See the [documentation](https://goreleaser.com/customization/homebrew_casks/)
for more details.

> [!NOTE]
> This feature was just recently introduced in Homebrew.

## Source RPMs

GoReleaser can now generate Source RPM (`.src.rpm`) packages!

This one has been a long time coming - the feature request dates back to 2022,
and it's finally here.
Huge thanks to [Tom Payne](https://github.com/twpayne) for working on it!

Source RPMs are how RPM-based distributions (Fedora, RHEL, CentOS, etc.)
package software for redistribution and rebuilding. They bundle the source
archive together with a `.spec` file that describes how to produce binary RPMs.

A minimal example:

```yaml {filename=".goreleaser.yaml"}
srpm:
  enabled: true
  spec_file: myproject.spec.tmpl
  summary: My project summary
  license: MIT
  url: https://myproject.example.com
```

See the [documentation](https://goreleaser.com/customization/package/srpm/) for
the full list of configuration fields and an example Fedora-style spec template.

## New website

![goreleaser.com](https://carlosbecker.com/posts/goreleaser-v2.15/img.png)

We migrated the documentation website from Material for MkDocs to
Hugo with the Hextra theme.

The new site is faster, and we took the opportunity to clean up and improve
the docs along the way.

If you run into any broken links, please let us know — or open a PR adding a
redirect to the `_redirects` file.

You can read the full announcement [here](https://goreleaser.com/blog/new-site/).

## Before publish installer types and SBOM support

{{< featpro >}}

The `before_publish` hook now works with NSIS and `.pkg` installer types, so
you can run scripts before those artifacts are published.

The SBOM pipe now covers installer types too — you can generate SBOMs for your
`.pkg` and NSIS installers alongside your binaries.

We also fixed a handful of rough edges left over from v2.14: NSIS and `.pkg`
artifacts are now correctly uploaded to releases and blob storage, play nicely
with the custom publisher, and are included when signing installers.

## Other updates

- [**checksums**](https://goreleaser.com/customization/checksum/): added BLAKE3 checksumming support
- [**telegram**](https://goreleaser.com/customization/telegram/): added `message_thread_id` to post to a specific supergroup thread; fixed response body not being closed
- [**gomod**](https://goreleaser.com/customization/gomod/): retry Go mod proxy fetches on `404` with exponential backoff
- [**homebrew_casks**](https://goreleaser.com/customization/homebrew_casks/): fixed stanza order; use heredoc for caveats to handle shell metacharacters
- [**docker**](https://goreleaser.com/customization/docker/): check if `--provenance` and `--sbom` flags are available before using them
- [**rust**](https://goreleaser.com/customization/rust/): support `cargo-zigbuild` targets with custom glibc versions
- [**go**](https://goreleaser.com/customization/build/): removed `windows/arm` from valid build targets
- [**release**](https://goreleaser.com/customization/release/): fixed `ignore_tags` filtering when multiple tags are set
- [**pro/changelog**](https://goreleaser.com/customization/changelog/): fixed changelog generation on the first release

## Other news

- GoReleaser now has ~15.7k stars and 458 contributors! Thanks, everyone!
- You can now follow release updates on our
  [Telegram channel](https://t.me/goreleasernews)!
- We often discuss new features in our Discord server.
  [Join the conversation][discord]!
- nFPM had new releases as well,
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.15.0" label="v2.15.0 (OSS)" icon="github" primary="false" >}}
{{< button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.15.0" label="v2.15.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}

[discord]: https://goreleaser.com/discord
[mkdocs]: https://squidfunk.github.io/mkdocs-material/
[hugo]: https://gohugo.io
[hextra]: https://imfing.github.io/hextra/
[redirs]: https://github.com/goreleaser/goreleaser/blob/main/www/static/_redirects
