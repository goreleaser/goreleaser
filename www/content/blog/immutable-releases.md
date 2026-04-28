---
title: Moving to Immutable Releases
date: 2026-04-26
slug: immutable-releases
tags:
  - announcements
  - security
authors:
  - caarlos0
---

GoReleaser is moving to immutable releases. From now on, no tag we publish
can ever be overwritten — once a version is out, it stays exactly as it was
published, forever.

<!--more-->

We should have enabled this a long time ago, but GitHub only allows to enable
immutable releases for all releases, and we were using a moving `nightly` tag
for the nightly releases.

To fix that, we needed to [update our action][action], and tune our
configuration a bit as well.

Starting now, nightly builds will get their own tags in the
`{next-minor}-{sha}-nightly` format.

For example, instead of pulling `nightly`, you'll see tags like
`v2.16.0-abc1234-nightly`. The previous moving `nightly` tag is still there, but
it will never be updated again, and might be deleted in the future.

## Why

Mutable tags are a supply-chain hazard. If the bytes behind a tag can change,
then a compromised release pipeline — ours or anyone else's — can silently swap
a known-good artifact for a malicious one, and every consumer pinning that tag
picks it up on the next pull. There is no way for a downstream user to detect
it short of hashing every download.

Immutable releases close that door. Once `vX.Y.Z` (or
`vX.Y.Z-sha-nightly`) is published, the contents are frozen. A release can't
be hijacked after the fact, and reproducing or auditing a specific build
becomes a matter of pinning a single version string.

This is also still a part of the [GitHub Open Source Secure Fund][ghossf]
initiative, as well as a request from [several users][request].

[action]: https://github.com/goreleaser/goreleaser-action/releases/tag/v7.2.0
[ghossf]: /blog/github-secure-fund/
[request]: https://github.com/goreleaser/goreleaser/discussions/6550
