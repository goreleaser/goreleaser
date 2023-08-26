---
date: 2023-04-10
slug: goreleaser-v1.17
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.17 — the late Easter release

The Easter release is here!

<!-- more -->

![better docker image build error output](https://carlosbecker.com/posts/goreleaser-v1.17/img.png)

It is packed with some juicy features and tons of bug fixes and quality-of-life
improvements.

Let's take a look:

### Highlights

- GoReleaser Pro now has a `templated_files` (or `templated_extra_files`) in
  several fields: archives, blobs, checksum, custom_publishers, docker, nfpms,
  release, snapcrafts and source
- GoReleaser can now open pull requests of homebrew taps, brews and scoop
  instead of just pushing it to a branch
- Some smaller improvements in templates, like the new `.Now` and `.IsDraft`
  template variables (tip: `{{ .Now.Format "2006" }}` to format the date time as
  you want)
- Default Parallelism now matches Linux container CPU
- Some errors have been improved to be more clear on how to fix them
- Many improvements in the documentation
- As always, a lot of bug fixes, dependency updates and improvements!

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.17.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.17.0

### Other news

- We have a whole lot of example repositories, including Zig, GoReleaser-Cross,
  GoReleaser Pro features, and more.
  [Check it out](https://github.com/orgs/goreleaser/repositories?q=example)!
- GoReleaser now has ~11.4k stars and 333 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
