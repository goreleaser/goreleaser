---
date: 2023-05-05
slug: goreleaser-v1.18
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.18 — the maintainers month release

May is the [maintainers month](https://maintainermonth.github.com), so I would
first like to thank all the maintainers out there for the hard work, you rock!

<!-- more -->

![new goreleaser -v output (pro)](https://carlosbecker.com/posts/goreleaser-v1.18/img1.png)

Now, onto new features!

## Highlights

### Native `upx` support

This version introduces a [`upx`][upx] root configuration section, which allows
you to compress the binaries.

Go built binaries are known for being, well, not small. There are a couple of
strategies to remediate it, for example, passing `-s -w` as `-ldflags` to `go
build` - which GoReleaser does by default since the beginning.

I hope that, by making it easier to make the binaries even smaller, we get more
projects to do that, better supporting environments with bad download speeds
and/or low storage.

For reference, running `upx` through GoReleaser binaries shrunk them from
**~51M** to **~15M** - about **27%** of its original size.

In a future release we'll also add more filters to the `upx` configuration.

[Documentation](https://goreleaser.com/customization/upx/).

PS: if you use this on GitHub Actions, I recommend using
[`crazy-max/ghaction-upx`](https://github.com/crazy-max/ghaction-upx) to install
the latest and greatest [`upx`][upx] version!

[upx]: https://upx.github.io/

### Report binaries sizes

Also related to binary sizes, you can now enable size reporting. After the build
phase, GoReleaser will display the sizes and paths of all built artifacts.

These sizes will also be available in `dist/artifacts.json`, so you might parse
and export them somewhere else.

[Documentation](https://goreleaser.com/customization/reportsizes/).

### Template improvements

This is a recurrent subject in most releases, I know: more templateable fields!
This release is not different in that regard:

- New `{{ .IsGitDirty }}` template variable
- `nfpms.*.package_name` now allows templates

### Scoops

To be in better parity with [`brews`][brews] and others, `scoop` is deprecated
in favor of `scoops`, and you can now define multiple Scoop manifests in the
same `.goreleaser.yaml` file.

[Documentation](https://goreleaser.com/customization/scoop/).

[brews]: https://goreleaser.com/customization/homebrew/

### Publish Homebrew taps, Scoop manifests and Krew plugins to plain Git repositores

Historically, you could only publish to GitHub, GitLab and Gitea, which used
their respective APIs.

Now, you can push to any Git repository. This can be specially useful for people
self-hosting Git servers, like [Soft Serve][soft] for example.

[soft]: https://charm.sh/soft-serve

Documentation:

- [Homebrew Taps](https://goreleaser.com/customization/homebrew/).
- [Scoops Manifests](https://goreleaser.com/customization/scoop/).
- [Krew Plugin Manifests](https://goreleaser.com/customization/krew/)

### Deprecation warnings rolled out

On GoReleaser Pro, the initial way to access custom environment variables was
`{{.var_name}}`. That could conflict with GoReleaser's internal state, and was
deprecated in favor of `{{.Var.var_name}}`.

Now, the old way is officially removed for good.

### Output improvements

I always aim for the GoReleaser output to be concise yet complete-ish.

This release contains a few improvements in that regard, like the removal of
sorting the `log` keys alphabetically, so they are displayed in the intended
order.

Another change is in printing the artifacts' path: it will now, when possible,
use relative paths in order to make the output a bit better.

Last but not least, we have a new `goreleaser --version` output using
[go-version](https://github.com/caarlos0/go-version):

{{< img caption="new goreleaser -v output" src="img2.png" >}}

It's not much, but it's honest work!

### Check multiple configuration files

GoReleaser Pro allows you to include configuration files, which might lead to
[repositories of reusable `.goreleaser.yaml` files
parts](https://github.com/caarlos0/goreleaserfiles).

One of the advantages of this is being able to, for example, change some
configuration in a single place and that is then applied to all the projects
that use that file.

The problem is that `goreleaser check` only ever allowed to check one file at a
time.

On v1.18 we added the ability to, instead, pass as many configuration files you
need as arguments. This also allows you to use shell globs, e.g.:

```bash
goreleaser check goreleaser*.yaml
```

You can then add this to your CI, so your GoReleaser configuration files are
always validated and hopefully free of deprecation notices. 🤝

### No AUR for v1.18.0

I'm not sure if AUR is under maintainance or if there something else going on,
but I'm unable to clone my AUR packages using their private URLs.

I didn't want to hold the release because of it, so I'll be releasing a patch as
soon as the problem is fixed, whatever the problem is.

Thanks for the comprehension. 😃

## Other news

- We have a whole lot of example repositories, including Zig, GoReleaser-Cross,
  GoReleaser Pro features, and more.
  [Check it out](https://github.com/orgs/goreleaser/repositories?q=example)!
- GoReleaser now has ~11.6k stars and 336 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.18.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.18.0
