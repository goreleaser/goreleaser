---
date: 2024-02-05
slug: goreleaser-v1.24
categories: [announcements]
authors: [caarlos0]
---

# Announcing GoReleaser v1.24 - the first of 2024

Happy new year!
The first release of 2024 is here!

<!-- more -->

## Highlights

- **security**: goreleaser would log environment variables in some
  configurations when run with `--verbose`. Note that we only recommend using
  the `--verbose` flag locally, to debug possible issues.
  [CVE-2024-23840](https://nvd.nist.gov/vuln/detail/CVE-2024-23840)
- **new**: create DMG images (with `hdutil`/`mkisofs`) (only on [Pro][pro])
- **new**: create MSI installers (with `wix`/`msitools`) (only on [Pro][pro])
- **git**: options to ignore tag prefixes (only on [Pro][pro])
- **blog**: we fully migrated our blog from Medium to [mkdocs](/blog)
- **blob**: ACLs, cache control, and content disposition
- **nfpm**: add libraries to packages
- **artifactory**: allow to publish source archives
- **brew**: improve handling of single OS
- **nix**: improved generated derivations, use `stdenvNoCC` by default
- **templates**: new fields added, `.GitTreeState`, `.IsGitClean`, `.Format`
- **jsonschema**: we now validate our jsonschema every time it changes to make
  sure it is still valid
- **deprecations**: we deprecated some options in the `changelog` and `blobs`
  sections. [Details](/deprecations)

As always, bug fixes, dependency updates, housekeeping, and documentation
updates are included in this release as well.

## Other news

- GoReleaser now has ~12.6k stars and 370 contributors! Thanks, everyone!
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
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.24.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.24.0
[discord]: https://goreleaser.com/discord
