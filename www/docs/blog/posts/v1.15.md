---
date: 2023-01-30
slug: goreleaser-v1.15
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.15 — the first of 2023

Keeping our pace of 1 minor a month, this is the January 2023 release.

<!-- more -->

![GoReleaser’s Ko integration documentation](https://carlosbecker.com/posts/goreleaser-v1.15/img.png)

It is packed with some juicy features and tons of bug fixes and quality-of-life
improvements.

Let's take a look:

### Highlights

- GoReleaser Pro now can now create changelog subgroups
- You can create and push Docker images/manifests with [Ko](https://ko.build/)
  (big thanks to [@developerguyba](https://twitter.com/developerguyba) and
  [@ImJasonH](https://twitter.com/ImJasonH) for all the work here)
- More templateable fields: `nfpms.apk.signature.key_name`, `release.disable`,
  `release.skip_upload`, `snaps.grade`, `telegram.chat_id`
- Deprecated `--clean` in favor of `--clean`
- As always, a lot of bug fixes and documentation improvements

You can [install][] the same way you always do, and you can see the full release
notes [here][oss-rel] and [here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.15.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.15.0

#### GitHub Action incident

Last Friday (Jan 27th), we had a rather big
[incident](https://github.com/goreleaser/goreleaser-action/pull/389) due to a
GitHub website change. We were using a URL to get the release that was not in
the API domain, but was working well for a couple of years. GitHub changed it,
and we, as well as many other projects, got an incident in our hands. This is
definitely our fault, though, we should have been using the guaranteed API
endpoints instead. That said, we fixed it rather quickly, even though it was a
Friday night. Since then, we also made more changes to use the `releases.json`
served by our website instead of GitHub’s API, which also avoids rate limits and
issues with GHE users. I also wanna give huge props to
[@crazy-max](https://github.com/crazy-max) for working hard on all that, as well
as everyone who helped debug and test everything on
[#389](https://github.com/goreleaser/goreleaser-action/pull/389).

### Other news

- GoReleaser now has ~11.2k stars and 322 contributors! Thanks, everyone!
- We are now present in the fediverse, give us a follow at
  [@goreleaser@fosstodon.org](https://fosstodon.org/@goreleaser).
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).
