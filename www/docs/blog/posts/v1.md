---
date: 2021-11-14
slug: goreleaser-v1
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1 — the first v1 release

Hello everyone!

I've been holding on the "v1" release for, _checks notes_, years now. That's
because I wanted v1 to have a "stable enough API", i.e. something unlikely to
change.

<!-- more -->

A couple of months ago I realized that we'll probably never reach that, as
things keep changing: we add more features, change old ones (sometimes on our
own, sometimes due to changes on other tools), and so on. That way, v1 would
never happen.

Therefore, after **467 releases**, **3299 commits**, and almost **5 years** of
hard work, I'm glad to announce that **v1.0.0 is finally here**!

![It's happening!](https://carlosbecker.com/posts/goreleaser-v1/51e877d4-5114-45e7-8cda-4aa9189701bd.gif)

This is a special release, as it marks GoReleaser departing from the school of
[ZeroVer](https://0ver.org), so I decided to write this quick bit compiling the
most notable changes and some special thanks.

## Most notable changes

- New `ReleaseURL` template variable, which points to the current tag release
  page on GitHub/Gitlab/Gitea
- Release [Krew Plugin Manifests](https://goreleaser.com/customization/krew/)
- [Announce releases to
  LinkedIn](https://goreleaser.com/customization/announce/linkedin/)
- Support [GitHub-generated release
  notes](https://goreleaser.com/customization/changelog/)
- Better support for [keyless
  signing](https://goreleaser.com/customization/sign/) (with `cosign` for
  example)
- Several [nFPM-related
  improvements](https://goreleaser.com/customization/nfpm/): "conventional file
  naming" per target, better arch handling, `dir` content type and others
- [Renamed master to
  main](https://medium.com/idealo-tech-blog/inclusive-language-in-tech-82b19b34b7cf)
  on both GoReleaser and nFPM
- Some bug fixing (as always)

If it feel like a regular GoReleaser v0.x release, it's because it is. The
biggest change here is that we're leaving v0! 😅

## Thanks

And a big **thank you** to everyone that helps with code, issues, money, support
and whatnot.

I really appreciate it! You all make OpenSource and Indie development a bit
easier! 💙

## Get it!

Without further ado, you can get it here:

- [GoReleaser v1.0.0](https://github.com/goreleaser/goreleaser/releases/tag/v1.0.0)
- [GoReleaser Pro v1.0.0](https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.0.0-pro)
