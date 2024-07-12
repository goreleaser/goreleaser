---
date: 2024-06-04
slug: goreleaser-v2
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v2

The new major version of [GoReleaser][gorel] is here!

<!-- more -->

![GoReelaser banner](https://carlosbecker.com/posts/goreleaser-v2/header.png)

I first launched [GoReleaser v1.0.0][v1] in November 2021 - 2.5 years ago!

The main reason for this is that so I can clean up old deprecated options, and
cleaning these old things makes it easier to add new things.

A couple of months ago I published a post explaining how GoReleaser's versioning
will work from now on.
You can read it [here][versioning].

So, after _two and a half years_, it's beyond time for v2 to happen, don't you
think?

![It's happening!](https://carlosbecker.com/posts/goreleaser-v2/happening.gif)

## Highlights

GoReleaser v2 is basically the same as the [GoReleaser v1.26.2][last-v1] - the
latest v1.
The differences between them should be only the removal of deprecated options.

[goreleaser --version](https://carlosbecker.com/posts/goreleaser-v2/pro.png)

All that being said, we do have a lot of new features since the [first v1][v1].
Here's an incomplete list with some of my favorites:

1. Allow to template entire files and use them in release/archives/etc (Pro)
1. Allow publishing of Nix, Brew, etc across SCMs (Pro)
1. Added the `changelog` command and changelog sub-grouping (Pro)
1. Split & merge releases (`release --prepare` and `continue`) (Pro)
1. Added SBOM creation support
1. Support Keyless signing with [Cosign][]
1. [Arch User Repository][aur] Support (aka AUR)
1. Added support for `GOAMD64` and `WASI`
1. Support creating [Chocolatey][] packages
1. [Ko][ko] support
1. Added the `healthcheck` command
1. Added more announcers: HTTP, Bluesky, Mastodon, etc
1. Allow to compress binaries with [upx][]
1. Added [Nix][nix] User Repository support (aka NUR)
1. Added [Winget][winget] support
1. Allow Homebrew, Krew, Scoop, Winget, etc to open pull requests
1. Added support for DMG creation
1. Added support for MSI creation
1. Added macOS binaries notarization and signing
1. A whole lot of improvements regarding templates: new variables, new fields,
   new functions. 😎

## Upgrading

If you keep up with the [deprecation notices][notices], it's likely you don't
need to do anything.

If you don't, that's fine too! Let's figure it out together!
You can start by running the following commands:

```sh
goreleaser check # using the latest v1
rm -rf ./dist/
grep -iR '\--rm-dist' .
grep -iR '\--skip-' .
grep -iR '\--debug' .
```

> **Extra tip**: You can also look into your last release logs if they are
> still there, and fix the deprecation warnings based on it.

Fix any occurrences following [this][notices], then, upgrade `goreleaser` to v2
using the method you used to install v1, and run:

```sh
goreleaser check # using latest v2
```

It should only warn you about the `version` header in the configuration file,
which you can fix by adding `version: 2` to it.

Then, you should be ready to use GoReleaser v2!
You can build a snapshot with:

```sh
goreleaser release --snapshot --clean
```

## GitHub Action

If you use our [GitHub Action][action], the latest version (v6.0.0) should
use `~> v2` by default if your `version` option is either empty or `latest`.

I recommend you update it:

```yaml
# .github/workflows/release.yml

# ...
jobs:
  goreleaser:
    # ...
    - uses: goreleaser/goreleaser-action@v6
      with:
        distribution: goreleaser # or 'goreleaser-pro'
        version: "~> v2" # or 'latest', 'nightly', semver
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        # GORELEASER_KEY: ${{ secrets.GORELEASER_KEY }} # if using goreleaser-pro
```

## Other news

- GoReleaser now has ~13.2k stars and 394 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  You are invited to [join the conversation][discord]!
- [goreleaser-action@v6](https://github.com/goreleaser/goreleaser-action/releases/tag/v6.0.0)
  was also released, now defaults to `version: '~> v2'` instead of `latest`.

---

That's all for today!

We start working on v2.1 **now**, and it should be released _soon-ish_. 👌

Happy releasing! 🚀

[action]: https://github.com/goreleaser/goreleaser-action
[versioning]: https://goreleaser.com/blog/release-cadence/
[gorel]: https://goreleaser.com
[upx]: https://upx.github.io
[Chocolatey]: https://chocolatey.org
[ko]: https://ko.build
[winget]: https://learn.microsoft.com/en-us/windows/package-manager/winget/
[nix]: https://nixos.org
[aur]: http://aur.archlinux.org
[Cosign]: https://github.com/sigstore/cosign
[last-v1]: https://goreleaser.com/blog/goreleaser-v1.26
[v1]: https://goreleaser.com/blog/goreleaser-v1
[discord]: https://goreleaser.com/discord
[notices]: https://goreleaser.com/deprecations/#removed-in-v2
