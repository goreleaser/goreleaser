---
date: 2023-11-06
slug: goreleaser-v1.22
categories: [announcements]
authors: [caarlos0]
---

# Announcing GoReleaser v1.22 - steady improvement

Another boring release, with mostly bug fixes and quality-of-life improvements.

<!-- more -->

## Highlights

- Several new pipes can be skipped with `--skip=pipe`, check `goreleaser release
--help` for details
- If you have `gomod.proxy` enabled, GoReleaser will now check if your
  `go.mod` has `replace` directives, and warn you about them on snapshots,
  and straight out fail on a production build
- If you have `gomod.proxy` enabled and a `go.work` file with multiple modules,
  GoReleaser will now properly handle it, using the first module as proxy
  target
- Planning for v2, we added an optional `version` field to the configuration
  file
- The checksum file will now be sorted by filename, as it should

As always, bug fixes, dependency updates, housekeeping, and documentation
updates are included in this release as well.

## Other news

- GoReleaser now has ~12.3k stars and 356 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation][discord]!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

## Helping out

You can help by contributing features and bug fixes, or by donating.
You may also be interested in buying a GoReleaser Pro license.

You can find out more [here](https://goreleaser.com/sponsors/).

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.22.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.22.0
[discord]: https://goreleaser.com/discord
