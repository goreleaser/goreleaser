---
date: 2022-01-26
slug: goreleaser-v1.4
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.4 — the AUR release

GoReleaser can now create and publish Arch Linux `PKGBUILD` files to Arch User
Repositories!

<!-- more -->

![neofetch in an arch linux container](https://carlosbecker.com/posts/goreleaser-v1.4/0fab21e9-bab6-4ba6-bd6a-c99b63c868a8.png)

This amazing new feature was sponsored by [Charm](https://charm.sh/).

## **How does it work?**

The AUR is basically a group of Git repositories that, if you push the right set
of files, are installable on an Arch Linux box using a tool
like [yay](https://github.com/Jguer/yay).

To push there, you'll need to create an account on 
[the AUR website](https://aur.archlinux.org/) and inform them of the public key
you'll be using to push.

Then you just need to 
[add some configuration to your goreleaser.yaml](https://goreleaser.com/customization/aur) 
file and...you're done! GoReleaser will build everything for you, clone the
repository, update the files, commit everything, and push it back.

**And that's it!** You can now instruct your users to install your package with
a command like `yay -S goreleaser-bin` or `yay -S goreleaser-pro-bin`. How
awesome is that!?

## **Other notable features**

- [GoReleaser Pro](https://goreleaser.com/pro) now allows you to
  use `-snapshot` without a `-key`. This should help users without a license key
  test things locally, or on a CI job with fewer privileges (e.g. GitHub Actions
  on a pull request).
- And finally, both the OSS and Pro distributions now have man pages: run `man
goreleaser` or `man goreleaser-pro` to check them out.
- On [GoReleaser Pro](https://goreleaser.com/pro), custom variables should now
  be  [prefixed with `.Var`](https://goreleaser.com/deprecations/#variables).

## **Other news**

- We still don't have a new date for our first community call. Personal life a
  little too busy lately, will try my best to schedule it
  ASAP. [Link](https://github.com/goreleaser/community/pull/2).
- GoReleaser now has ~9.5k stars and 262 contributors! Thanks, everyone!
- Our Discord server is getting new members almost daily. 
  [Join and chat with us](https://discord.gg/RGEBtg8vQ6)!

---

Full disclosure: [Charm](https://charm.sh/) is my current employer.

Shameless plug: Definitely [check us out](https://charm.sh/), we are 
[building a bunch of cool OSS stuff](https://github.com/charmbracelet)!
