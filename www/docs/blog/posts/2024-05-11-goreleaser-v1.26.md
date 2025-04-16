---
date: 2024-05-11
slug: goreleaser-v1.26
categories: [announcements]
authors: [caarlos0]
---

# Announcing GoReleaser v1.26 - The last v1, probably

Happy mother's day!

This will be probably the last minor v1 release of GoReleaser.
V2 will not be a big update, rather, it'll be the same as v1.26, but removing
all the deprecated stuff.

That said, let's see what's new on this version!

<!-- more -->

## Highlights

- **new**: macOS binaries notarization
- **new**: Publish Homebrew, NUR, Winget, and others, across SCMs (only on [Pro][pro])
- **fury**: Retry uploads
- **continue**: Fixed `goreleaser continue --merge` when running with
  `--snapshot`
- **announce**: BlueSky support
- **archive**: Create `.tar.zst` archives
- **changelog**: Allow to customize the changelog format
- **checksum**: Support Blake2 and SHA-3
- **release**: Added `--draft` to `goreleaser release`
- **gitea**: Support changelog
- **gitlab**: Support opening Merge Requests
- **build**: Always log `go build` outputs
- **tmpl**: Added `isEnvSet`
- **homebrew**: Updated to use `on_arm` and `on_intel`
- **blob**: Allow to skipping the configuration of the `Content-Disposition`

As always, bug fixes, dependency updates, housekeeping, and documentation
updates are included in this release as well.

## Other news

- GoReleaser now has ~13.1k stars and 380 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation][discord]!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
- In preparation for v2, the
  [v5.1.0 of our GitHub Action](https://github.com/goreleaser/goreleaser-action/releases/tag/v5.1.0)
  now defaults to `version: '~> v1'` instead of `latest`.
  This should help prevent unwanted breaking changes.
  [More details](https://github.com/goreleaser/goreleaser-action/pull/461).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also [sponsor the project](/sponsors), or get a
[GoReleaser Pro license][pro].

[pro]: /pro
[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.26.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.26.0
[discord]: https://goreleaser.com/discord
