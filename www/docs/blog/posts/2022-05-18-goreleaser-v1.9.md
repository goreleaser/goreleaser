---
date: 2022-05-18
categories:
  - announcements
authors:
  - caarlos0
slug: goreleaser-v1.9
---

# Announcing GoReleaser v1.9 — the 10k stars release

This release contains several minor improvements and a couple of new features! Let's have a look!

<!-- more -->

![goreleaser changelog preview](https://carlosbecker.com/posts/goreleaser-v1.9/136319f0-2346-4c3b-aa77-e239a7563527.png)

## **Highlights**

- `goreleaser changelog` was added to [GoReleaser Pro](https://goreleaser.com/pro) — you can use it to preview your next release changelog
- added more build options, enabling you to build test binaries [#3064](https://github.com/goreleaser/goreleaser/pull/3064)
- added `go_first_class` target options for build [#3062](https://github.com/goreleaser/goreleaser/pull/3062)
- new run script for CIs that don't have it natively [#3075](https://github.com/goreleaser/goreleaser/pull/3075)
- make it easier to run GoReleaser against a SCM that is not GitHub, GitLab or Gitea [#3088](https://github.com/goreleaser/goreleaser/pull/3088)
- allow to create meta archives [#3093](https://github.com/goreleaser/goreleaser/pull/3093)
- the archive pipe no longer check links are valid, like `tar` [#3103](https://github.com/goreleaser/goreleaser/pull/3103)
- a lot of bug fixes and docs improvements
- we are also experimenting with new ways to share news with our users, the latest is our [newsletter](https://www.getrevue.co/profile/goreleaser)

You can see the full changelog [here](https://github.com/goreleaser/goreleaser/releases/tag/v1.9.0).

## **Other news**

![We hit 10k stars!](https://carlosbecker.com/posts/goreleaser-v1.9/23d24a10-64ad-4fbd-aa37-d7e0367fe9d9.png)

- Still no community meeting, sorry 🫠. [Link](https://github.com/goreleaser/community/pull/2).
- We hit 10k stars on our [main repository](https://github.com/goreleaser/goreleaser)!
- GoReleaser now has ~10.1k stars and 277 contributors! Thanks, everyone!
- Our Discord server is getting new members almost daily. [Join and chat with us](https://discord.gg/RGEBtg8vQ6)!
