---
date: 2022-11-18
slug: goreleaser-v1.13
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.13 — the November release

Another month, another release!

Like the previous 2 releases, this is a beefy one: over 100 commits from 15
contributors!

<!-- more -->

![Mastodon integration](https://carlosbecker.com/posts/goreleaser-v1.13/picture.png)

This one also marks the point of
[1 year since our first v1](./v1.md)!

### Highlights

- `docker`, `docker buildx` and `podman` (on GoReleaser Pro) will now use the
  image `digest` when creating `docker manifests`. This should help ensure that
  what you are releasing wasn't changed by an outside tool.
  - In the same token, the Docker images and manifests signing with `cosign`
    will now use the `digest` by default.
- GoReleaser can now announce to Mastodon!
- Ability to create Arch Linux packages, _btw_.
  - Still regarding nFPM, `dst`s with trailing slashes will now have the same
    behavior as tools such as `cp` with trailing slashes.
- For Windows fans: you can now create and publish `nupkg`s (Chocolatey
  packages)!
- Better support for building and publishing shared or static libraries.
- Many bug fixes and documentation improvements.

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.13.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.13.0

### Other news

- GoReleaser now has ~10.9k stars and 313 contributors! Thanks, everyone!
- We are now present in the fediverse, give us a follow at
  [@goreleaser@fosstodon.org](https://fosstodon.org/@goreleaser).
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
- We made a bunch of progress towards getting 100% in the
  [OpenF Best Practices](https://bestpractices.coreinfrastructure.org/en/projects/5420#analysis)
  assessment... and we're almost there.
