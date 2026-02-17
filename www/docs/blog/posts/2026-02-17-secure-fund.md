---
date: 2026-02-17
slug: github-secure-fund
categories: [announcements]
authors: [caarlos0]
---

# GoReleaser participation on GitHub's Secure OpenSource Fund

GoReleaser was selected to participate in the third session of the
[GitHub Secure OpenSource Fund][ghosf].

<!-- more -->

We joined a group with [so many great projects][announcement] that I feel bad
trying to name just a few of them - so you should check the official
announcement for the full list!

That said, in that session we did a **lot** of improvements in GoReleaser, just
to name a few:

- Better documentation and understanding of attack surface (e.g. IRP, Threat Modeling, etc)
- Using SARIF for all security scanners - and added more of them
- Improved GitHub Actions usage to be more secure
- Reviewed dependencies
- Reviewed and improved SBOMs
- And much more!

Granted, we were already doing some things right, mostly thanks to the feedback
of our amazing community.
For instance, we had signing, SBOMs, and private vulnerability reports for a
long time.

Granted, there's always room for improvement!

## Go is a solid ~~language~~ platform, actually

Go has been a solid foundation for GoReleaser, from the early days of only
supporting Go ourselves, to now, supporting multiple languages.

For instance, Go has fuzzing built-in, which is something we definitely want to
take more advantage of in the coming months.

[`govulncheck`][govulncheck] is amazing as well, and every Go-based project
should run it as part of their pipeline.

By the way, if you want to make your project a bit more secure, check out
[this secure example repository][example]: it is using a lot of the good
practices learned during the session, as well as some particularities of
GoReleaser itself.

## Looking forward

Security work is never really done.
There's a long road ahead, but I feel like we are way more secure now than
before.

Our greatest thanks to both the fund and the fund's partners for this
opportunity for everything.

[ghosf]: https://github.com/open-source/github-secure-open-source-fund
[example]: https://github.com/goreleaser/example-secure
[govulncheck]: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck

<!-- TODO: fix link -->

[announcements]: https://github.blog/open-source/maintainers/securing-the-supply-chain-at-scale-starting-with-71-important-open-source-projects/
