---
date: 2023-12-26
slug: goreleaser-v1.23
categories: [announcements]
authors: [caarlos0]
---

# Announcing GoReleaser v1.23 - the last of 2023

The yearly Christmas edition, and the last release of 2023.
This release contains mostly small improvements and bug fixes.

<!-- more -->

## Highlights

- nix: validate license to prevent generating invalid derivations
- nix: make sure zip is included if one of the archives is a zip file
- winget: support `archives.format: binary`
- homebrew: `dependencies` can be added to specific OSes
- homebrew: support `tar.xz`
- aur: support `archives.wrap_in_directory`
- aur: support multiple packages in the same repository
- `--single-target` is now more consistent
- error handling improvements in several places

As always, bug fixes, dependency updates, housekeeping, and documentation
updates are included in this release as well.

## Other news

- GoReleaser now has ~12.4k stars and 366 contributors! Thanks, everyone!
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
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.23.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.23.0
[discord]: https://goreleaser.com/discord
