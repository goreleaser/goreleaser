---
date: 2022-03-06
slug: goreleaser-v1.6
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.6 — the boring release

GoReleaser 1.6 is out! Another "boring" release with some miscellaneous improvements and bug fixes.

<!-- more -->

## **Highlights**

- New `filter` and `reverseFilter` template functions ([#2924](https://github.com/goreleaser/goreleaser/pull/2924))
- nFPM and archiving in `tar.gz` should now be faster ([#2940](https://github.com/goreleaser/goreleaser/pull/2940), [#2941](https://github.com/goreleaser/goreleaser/pull/2941))
- More Snapcraft app metadata fields ([#2955](https://github.com/goreleaser/goreleaser/pull/2955))
- New `.TagBody` template field ([#2923](https://github.com/goreleaser/goreleaser/pull/2923))
- Install `amd64` binaries when no `arm64` binaries are present on macOS (i.e. use Rosetta) ([#2939](https://github.com/goreleaser/goreleaser/pull/2939))
- Several dependency updates
- Several bug fixes
- Some documentation improvements

You can see the full changelog [here](https://github.com/goreleaser/goreleaser/releases/tag/v1.6.0).

## **Other news**

- I'm risking sounding repetitive here, but we still don't have a new date for our first community call. Personal life a little too busy lately, will try my best to schedule it ASAP. [Link](https://github.com/goreleaser/community/pull/2).
- GoReleaser now has ~9.7k stars and 268 contributors! Thanks, everyone!
- Our Discord server is getting new members almost daily. [Join and chat with us](https://discord.gg/RGEBtg8vQ6)!
- nFPM also had a release as well, [check it out](https://github.com/goreleaser/nfpm/releases).
