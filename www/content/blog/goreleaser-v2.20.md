---
title: Announcing GoReleaser v2.20
date: 2026-11-04
slug: goreleaser-v2.20
tags: [announcements]
authors: [caarlos0]
---

GoReleaser v2.20 fixes a security issue that could expose inline SSH private
keys in release metadata. It also adds conditional ko platforms, better
grouping of GitLab release assets, and a Homebrew cask cleanup declaration.

<!--more-->

## Security fix: SSH private keys in release metadata

If you supplied an inline SSH private key for publishing package manifests,
including through an environment-variable template, GoReleaser could write
that key in cleartext to `dist/artifacts.json`.

Anyone with access to that file could recover the key and use its permissions
on the destination Git repository. This matters if you uploaded or shared
your `dist/` directory, for example as a CI artifact.

GoReleaser v2.20 excludes private keys from JSON output. Upgrade before your
next release. If you shared metadata containing a key, revoke and replace
that key, and remove the exposed copies.

See [GHSA-9422-8p9m-393x](https://github.com/goreleaser/goreleaser/security/advisories/GHSA-9422-8p9m-393x)
for details.

## ko platform templates

`kos.platforms` now accepts templates, so you can select image platforms based
on the release:

```yaml {filename=".goreleaser.yaml"}
kos:
  - repositories:
      - ghcr.io/myuser/myapp
    platforms:
      - linux/amd64
      - "{{ if not .Prerelease }}linux/arm64{{ end }}"
```

This builds both platforms for stable releases and only `linux/amd64` for
prereleases. Empty platforms are ignored; if none remain, the image is skipped.

Thanks to [Manuel Rüger](https://github.com/mrueg) for contributing this.

See the [documentation](https://goreleaser.com/customization/package/ko/)
for more details.

## GitLab release asset types

GitLab groups release assets by link type. GoReleaser now marks archives,
binaries, Linux packages, and other package-like artifacts as `package`.
Checksums, signatures, SBOMs, and other supporting files stay under `other`.

This applies to GitLab releases without a new setting:

```yaml {filename=".goreleaser.yaml"}
release:
  gitlab:
    owner: mygroup
    name: myapp
```

Thanks to [Manuel Vogel-Johnson](https://github.com/mavogel) for contributing this.

See the [documentation](https://goreleaser.com/customization/publish/scm/gitlab/#asset-link-types)
for more details.

## Homebrew casks without extra cleanup

When a cask has no `zap` directives, GoReleaser now writes `zap :not_required`
to declare that no extra cleanup is needed:

```yaml {filename=".goreleaser.yaml"}
homebrew_casks:
  - name: myapp
    repository:
      owner: myuser
      name: homebrew-tap
```

Existing `zap` directives are unchanged. The generated `zap :not_required`
stanza requires Homebrew 7.0.11 or later.

Thanks to [Simon Lacroix](https://github.com/SimonMiaou) for contributing this.

See the [documentation](https://goreleaser.com/customization/publish/homebrew_casks/)
for more details.

## Other updates

- [**changelog**](https://goreleaser.com/customization/publish/changelog/): skip changelog generation with `--skip=changelog`
- [**winget**](https://goreleaser.com/customization/publish/winget/): update the manifest schema to 1.28.0
- [**templates**](https://goreleaser.com/customization/general/templates/): escape backslashes in `mdv2escape`
- [**bluesky**](https://goreleaser.com/customization/announce/bluesky/): do not add a link facet when there is no release URL

## Other news

- GoReleaser now has ~16.1k stars and 512 contributors! Thanks, everyone!
- Join the conversation on [Discord][discord].
- You can follow release updates on our
  [Telegram channel](https://t.me/goreleasernews).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< g_button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.20.0" label="v2.20.0 (OSS)" icon="github" primary="false" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.20.0" label="v2.20.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< g_button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< g_button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}

[discord]: https://goreleaser.com/discord
