---
date: 2022-10-17
slug: goreleaser-v1.12
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.12 — the more-than-a-hundred commits release

[The previous release had ~100 commits](./v1.11.md),
and this one has 149 since previous feature release!

<!-- more -->

Definitely a big release, with some big features. Let's dive in!

![GoReleaser Pro release using the new spli/merge feature](https://carlosbecker.com/posts/goreleaser-v1.12/picture.png)

### Highlights

- **GoReleaser Pro** can now split and merge releases;
- **GoReleaser Pro** can now filter paths for changelogs;
- **GoReleaser Pro** has now a `continue` command, which merges `publish` and
  `announce`;
- **GoReleaser Pro** can now filter targets by `GGOOS` and `GGOARCH` as well;
- AUR can now set the backup options;
- GoReleaser completions are now published to Fig as well;
- Blobs have more templateable fields;
- The Telegram announcer now supports markdown;
- NFPMs can now create iOS packages (for jailbroken iPhones only);
- Buildpacks were permanently removed;
- As always, **a lot** of bug fixes, documentation improvements and dependencies
  updates;

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.12.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.12.0

### Other news

- GoReleaser now has ~10.8k stars and 305 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://discord.gg/RGEBtg8vQ6)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
