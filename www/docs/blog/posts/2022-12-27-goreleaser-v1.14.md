---
date: 2022-12-27
slug: goreleaser-v1.14
tags: [goreleaser, goreleaser-pro, golang]
---

# Announcing GoReleaser v1.14 — the Christmas release

Another month, another release!
In fact, the last release of the year.

<!-- more -->

This one in particular marks the 6 years anniversary of GoReleaser, and is
packed with new features and improvements.

![Santa](https://carlosbecker.com/posts/goreleaser-v1.14/img.png)

Let's see what's new:

### Highlights

- GoReleaser Pro can now skip the build of specific Docker images based on a
  template evaluation result;
- GoReleaser Pro build hooks now also inherit the build environment variables
- You can now use templates in `brews.install`, `builds.env` and
  `archives.files.info`
- Windows is added as a default OS for builds (amd64 and 386)
- New `archives.rlcp` option: It'll be the default soon, run `goreleaser check`
  to verify your configuration
- Allow to customize the tag sorting directive
- Deprecate `archives.replacements`
- Allow to set the file info of binaries inside archives
- Added a new `title` template function
- As always, a lot of bug fixes and documentation improvements

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.14.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.14.0

### Other news

- GoReleaser now has ~11k stars and 316 contributors! Thanks, everyone!
- We are now present in the fediverse, give us a follow at
  [@goreleaser@fosstodon.org](https://fosstodon.org/@goreleaser).
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
