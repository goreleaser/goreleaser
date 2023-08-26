---
date: 2022-02-12
slug: goreleaser-v1.5
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.5 — the misc improvements release

GoReleaser 1.5 is out, with a handful of miscellaneous improvements.

<!-- more -->

## **Highlights**

- Better manpages using [mango](https://github.com/muesli/mango);
- Migrated from [cobra](https://github.com/spf13/cobra) to [coral](https://github.com/muesli/coral) — which will eventually lead to faster `go install`;
- Improved nFPM to make it easier for GoReleaser's debs and GoReleaser-generated debs that pass [lintian](https://lintian.debian.org/) checks;
- Several improvements on GoReleaser output logs;
- More fields are now templateable, namely on nFPMs and Universal Binaries configs;
- Hooks now have an option to always print their outputs;
- `goreleaser build --single-target` now copies the binary to `CWD`, also accepts a `-output` flag telling to copy the binary elsewhere;
- Changelog passing through `goreleaser release --release-notes` now warns if the file is empty or whitespace-only, allowing to more easily debug releases;
- You can now override build `tags`, `ldflags`, `gcflags` and `asmflags` per target platform;
- On a similar note, you can get the runtime `GOOS` and `GOARCH` on template variables using `{{ .Runtime.Goos }}` and `{{ .Runtime.Goarch }}`.

You can see the full changelog [here](https://github.com/goreleaser/goreleaser/releases/tag/v1.5.0).

## **Other news**

- We still don't have a new date for our first community call. Personal life a little too busy lately, will try my best to schedule it ASAP. [Link](https://github.com/goreleaser/community/pull/2).
- GoReleaser now has ~9.6k stars and 264 contributors! Thanks, everyone!
- Our Discord server is getting new members almost daily. [Join and chat with us](https://discord.gg/RGEBtg8vQ6)!
- nFPM also had a couple of releases the last few weeks, [check them out](https://github.com/goreleaser/nfpm/releases).
