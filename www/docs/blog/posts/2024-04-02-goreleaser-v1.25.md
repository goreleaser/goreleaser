---
date: 2024-04-02
slug: goreleaser-v1.25
categories: [announcements]
authors: [caarlos0]
---

# Announcing GoReleaser v1.25 - Easter edition

Happy Easter!

The second release of 2024 is here!
It's the result of 2 months of work by me and many contributors, aiming on
releasing a v2 soon.

<!-- more -->

## Highlights

- **new**: automatically update the description/readme on Docker Hub (only on [Pro][pro])
- **new**: added `goreleaser verify-license` command (only on [Pro][pro])
- **archives**: allow to skip archiving certain `GOOS` by using `none` in
  `format_overrides`
- **dmg**: fix packaging when binary name contains a directory
- **msi**: fix packaging when binary name contains a directory
- **homebrew**: allow to set headers in an URL
- **homebrew**: sync fork before opening PR
- **krew**: sync fork before opening PR
- **nix**: sync fork before opening PR
- **scoop**: sync fork before opening PR
- **winget**: sync fork before opening PR
- **nix**: update valid licenses with upstream
- **nfpm**: signing passphrase improvements, support for compression, fieldsn
  and predepends on debs
- **git**: retry clone if possible
- **release**: mark release as a draft, upload all artifacts, then publish it
- **release**: allow to delete previously existing artifacts
- **checksums**: allow to create one checksum file for each published artifact
- **config**: look into `.config/goreleaser.ya?ml` by default
- **build**: support netbsd/arm64
- **release**: support project ID in GitLab
- **build**: support `directory` in `gomod`
- **deprecations**: a lot of deprecations, working towards making the
  configuration file more consistent. [Details](/deprecations)

As always, bug fixes, dependency updates, housekeeping, and documentation
updates are included in this release as well.

## Other news

- GoReleaser now has ~12.9k stars and 380 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation][discord]!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

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
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.25.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.25.0
[discord]: https://goreleaser.com/discord
