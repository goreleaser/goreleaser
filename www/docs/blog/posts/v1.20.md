---
date: 2023-08-09
slug: goreleaser-v1.20
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.20 — a quality-of-life release

A little over 100 commits in small-_ish_ quality-of-life improvements.

<!-- more -->

This is not a flashy release, but I bet you're going to like it anyway 😄

## Highlights

![Releasing...](https://carlosbecker.com/posts/goreleaser-v1.20/pic.png)

### Nightlies

I've been releasing [GoReleaser Pro Nightlies][pro-nightly] for a while now, but
it never had a fixed schedule, and the OSS version never had a nightly release
either.

Starting now, the Pro Nightly will be released every Wednesday, and the OSS
every Thursday.

[pro-nightly]: https://github.com/goreleaser/goreleaser-pro/releases/tag/nightly

### GoReleaser Pro improvements

[GoReleaser Pro][pro] added a few new features:

- ability to automatically check boxes in PR templates
- alternative names for Homebrew formulas
- `Dockerfile` templated contents
- HTTP & Artifactory upload matrix
- nFPM `templated_scripts`
- `goreleaser release --single-target`
- Release's footer/header can be set to file paths/URLs in the configuration file

### Nix

[We added Nix support in the previous release](./v1.19.md),
and in this one we added a few improvements:

- `zip` support
- the ability to define runtime dependencies
- make it easier to extend with a new `extra_install` option

### `extra_install`

Speaking of extra install instructions, we added this option to brew too.

### `mod_timestamp`

We added the ability to set a `mod_timestamp` to both metadata files and to
universal binaries.

### Other improvements and bug fixes

This release also adds a few other small improvements, here's a few of them:

- Scoops now support `arm64`
- Winget got the `PortableCommandAlias` option
- Release on GitHub now has the `make_latest` option
- You can now disable custom publishers using templates
- `goreleaser init` and overall `goreleaser release` output improvements

Make sure to read the [full release notes][oss-rel], and the
[pro version release notes][pro-rel] as well.

As always, we also had a bunch of bug fixes and documentation improvements.

## Other news

- GoReleaser now has ~11.9k stars and 348 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.20.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.20.0
[pro]: https://goreleaser.com/pro

## Helping out

You can help by contributing features and bug fixes, or by donating.
You may also be interested in buying a GoReleaser Pro license.

You can find out more [here](https://goreleaser.com/sponsors/).
