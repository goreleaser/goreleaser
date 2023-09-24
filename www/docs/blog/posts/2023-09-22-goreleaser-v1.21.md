---
date: 2023-09-22
slug: goreleaser-v1.21
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.21 - mostly bug fixes

A boring release, mostly bug fixes.
Boring is good.

<!-- more -->

## Highlights

- You can now sort tags by `semver` in [GoReleaser Pro][pro]
- Docker pushes will now be retried when the registry yields a 503. It'll retry
  10 times.
- Winget: added support for `package_dependencies` and update schema version to
  1.5.0.
- GoReleaser will now run against Gerrit, Soft-Serve, and other Git providers,
  as long as the SCM release is disabled.
- You can now ignore Git tags that match a regular expression.
- You can now skip pre build hooks on `goreleaser build`.
- Properly `go mod` handling on pre-mods Go versions.
- WASI support.
- New templateable fields and functions.
- Several bug fixes, specially when the runtime OS is Windows.

Besides that, some important refactories that should help evolving GoReleaser
further:

- `--skip` merges all the `--skip-*` flags, and will be extended to more
  features (open an issue requesting what you need 📩).
- Template error handling was improved.
- We also updated GoReleaser to Go 1.21.

And, as always, several bug fixes and documentation updates!

## Windows

GoReleaser was never properly/extensive tested on Windows.
It somewhat works, but there are many rough edges, and potentially some bugs
here and there.

To make Windows a first-class runtime for GoReleaser,
[I started trying to make CI pass on Windows](https://github.com/goreleaser/goreleaser/pull/4293).
It's a bit hard to evolve it as I don't really use Windows, and end up having
to juggle around VMs and/or CI.

If you use Windows regularly and want to contribute to OSS, ping me on our
[Discord][discord], I'm happy to help you in any way you need, especially if
you are a beginner in the field and/or Go.

## Blog

Our blog was migrated to a new address (and out of medium)!


You can read the announcement [here](./2023-09-14-welcome.md).

It still a work in progress, though.
You can also see the still-open issue
[here](https://github.com/goreleaser/goreleaser/issues/3503)

The old address will still be up indefinitely, but we won't be updating it any
longer.

## Other news

- GoReleaser now has ~12.1k stars and 352 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation][discord]!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

## Helping out

You can help by contributing features and bug fixes, or by donating.
You may also be interested in buying a GoReleaser Pro license.

You can find out more [here](https://goreleaser.com/sponsors/).

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.21.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.21.0
[pro]: https://goreleaser.com/pro
[discord]: https://goreleaser.com/discord
