---
date: 2022-07-04
categories:
  - announcements
authors:
  - caarlos0
slug: goreleaser-v1.10
---

# Announcing GoReleaser v1.10 — the summer release

Fun fact: it is actually winter now here in Brazil. Regardless, this release is packed with new features, quality-of-life improvements and bug fixes!

<!-- more -->

![GoReleaser Screenshot](https://carlosbecker.com/posts/goreleaser-v1.10/488e0554-ebcb-4e4f-90ed-a0c2c8b36fde.png)

### Highlights

- **GoReleaser Pro** can now skip global after hooks with `-skip-after`;
- **GoReleaser Pro** can now split the release process into "prepare", "publish" and "announce". Check out `goreleaser release --help`,`goreleaser publish --help` and `goreleaser announce --help` for more details;
- The entire GoReleaser output is now using [caarlos0/log](https://github.com/caarlos0/log), which is a more-or-less drop-in [apex/log](https://github.com/apex/log) replacement. It uses [Charm's technology](https://charm.sh/) for its implementation and has a slightly different proposal and feature set;
- New `-skip-docker` and `-skip-before` flags added;
- `goreleaser build` now allows repeatable `-id` filters;
- The build process now uses 1 parallelism permit for each binary being built, regardless of how many build configurations you have;
- GoReleaser can now load a config file from `STDIN` (using `-config -`);
- Builds can now override `env` in for a given target;
- `release.repo.owner` and `release.repo.name` can now be templated;
- Changelog grouping is now processed in the order they are declared and rendered in the order of their `order` field;
- GoReleaser now logs the duration of "slow" pipes;
- The deprecated `nfpms.empty_folders` is now removed;
- The deprecated handling of Windows ARM64 builds on Go versions older than 1.17 is now removed;
- As of every release, a good amount of bug fixing;
- And a bunch of documentation improvements.

### Other news

- GoReleaser now has ~10.3k stars and 285 contributors! Thanks, everyone!
- Our Discord server is getting new members almost daily. [Join and chat with us](https://discord.gg/RGEBtg8vQ6)!
- Check out a preview of the split release phases feature:

<iframe width="560" height="315" src="https://www.youtube-nocookie.com/embed/G0MF0R_LD1g?si=zunwxqAhjU6QFK9d" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
