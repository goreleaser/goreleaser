---
slug: goreleaser-v1.11
date: 2022-08-28
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.11 — the hundred commits release

This release took a while, for all the good reasons: a ton of new features and
bug fixes for your delight!

<!-- more -->

Oh, and, over 100 commits!

It might be the biggest GoReleaser release in commits made, although I have no
data to back it up — except my memory.

![GoReleaser Screenshot](https://carlosbecker.com/posts/goreleaser-v1.11/picture.png)

### Highlights

- **GoReleaser Pro** can now skip Fury publishing with `--skip-fury`;
- **GoReleaser Pro** now has before and after hooks for archives;
- GoReleaser is now compiled using Go 1.19, and supports new Go 1.19 targets
  (`GOARCH=loong64`);
- New `.ArtifactExt` template field and new `split` function;
- You can now add more files/folders/globs to source archives (e.g. the `vendor`
  folder);
- The JSONSchema is finally (properly) fixed;
- Skip uploading artifacts into the release, without skipping the whole release;
- Changelogs using the `github` strategy now use the short commit as well;
- Allow to keep a single draft GitHub release;
- Allow to set `target_commitish` in GitHub releases;
- Allow to set up mTLS in the HTTP uploads pipe;
- Option to strip the binary parent folder in the archive;
- Couple of improvements in the nFPM: added Termux packaging, changelogs and
  `provides` support.
- The GoReleaser Docker image now logs in into the GitLab Registry if its
  environment variables are set;
- Homebrew taps can now define their dependencies' versions;
- The deprecated Gofish feature is now fully removed;
- As of every release, a healthy amount of bug fixing;
- Many documentation improvements.

### Other news

- GoReleaser now has ~10.5k stars and 292 contributors! Thanks, everyone!
- GoReleaser now has a [LinkedIn page](https://www.linkedin.com/company/goreleaser/);
- We eventually discuss new features in our Discord server. [Join the conversation](https://discord.gg/RGEBtg8vQ6)!
- nFPM had new releases as well, [check it out](https://github.com/goreleaser/nfpm/releases).
- GoReleaser Pro now has [nightly releases](https://github.com/goreleaser/goreleaser-pro/releases/tag/nightly);
