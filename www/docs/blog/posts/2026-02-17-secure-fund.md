---
date: 2026-02-17
slug: github-secure-fund
categories: [announcements]
authors: [caarlos0]
---

# How GoReleaser strengthened security through GitHub's Secure Open Source Fund

GoReleaser builds and ships release artifacts for thousands of projects, making
it a high-value supply-chain target.
That's why we were thrilled to be selected for the third session of the
[GitHub Secure Open Source Fund][ghosf].

<!-- more -->

![](http://carlosbecker.com/posts/goreleaser-github-secure-oss-fund/img.png)

We joined a group with so [many great projects][announcement] that I feel bad
trying to name just a few of them - so you should check the official
[announcement][] for the full list!

That said, in that session we did a **lot** of improvements in GoReleaser.
Just to name a few:

- Better documentation and understanding of attack surface (e.g. IRP, Threat Modeling, etc)
- Wrote a lot of security-related documentation
- Using SARIF for all security scanners - and added more of them
- Improved GitHub Actions usage to be more secure
- Reviewed dependencies
- Reviewed and improved SBOMs
- Using OIDC to publish NPM packages
- And many more!

Granted, we were already doing some things right, mostly thanks to the feedback
of our amazing community.
For instance, we had signing, SBOMs, and private vulnerability reports for a
long time.

Still, there's always room for improvement!

## Go is a solid platform, actually

One interesting thing I realized whilst talking with other maintainers is that
Go has amazing tools for security.

For instance:

1. Go has fuzzing built-in, which is something we definitely want to
   take more advantage of in the coming months.
1. [`govulncheck`][govulncheck] is amazing as well, and every Go-based project
   should run it as part of their pipeline.
1. SBOMs can be easily generated with off-the-shelf tools like [`syft`][syft].

While some projects had to use external dependencies or write custom software to
do some of these things, with Go, it was really easy!

By the way, if you want to make your GoReleaser-powered project a bit more
secure, check out [this secure example repository][example]: it is using many
of the good practices learned during the session.

## Looking forward

Security work is never really done.
There's a long road ahead, but I feel like we are way more secure now than
before.

If you are interested in security, or just want to help, I'm always available on
the [GoReleaser Discord](/discord) - feel free to chime in there and let's chat. 🙏
[GitHub discussions][discussions] are also open.

Our greatest thanks to both the fund and the fund's partners for making this
possible.

[discussions]: https://github.com/orgs/goreleaser/discussions/new?category=ideas-issue-triage-and-general-discussions
[ghosf]: https://github.com/open-source/github-secure-open-source-fund
[example]: https://github.com/goreleaser/example-secure
[govulncheck]: https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck
[syft]: https://github.com/anchore/syft
[announcement]: https://github.blog/open-source/maintainers/securing-the-ai-software-supply-chain-security-results-across-67-open-source-projects/
