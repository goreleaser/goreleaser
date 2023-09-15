---
date: 2023-06-28
slug: goreleaser-v1.19
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.19 — the big release

Almost 200 commits adding Nix, Winget, and much more...

<!-- more -->

This release took almost **2 months** (!), and I hope the wait was worth it!

Without further ado, let's dive in!

## Highlights

### Security improvements

We got a [CVE on nFPM](https://github.com/goreleaser/nfpm/security/advisories/GHSA-w7jw-q4fg-qc4c)
and another one [on
GoReleaser](https://github.com/goreleaser/goreleaser/security/advisories/GHSA-2fvp-53hw-f9fc).

It is unlikely that you were affected by this, but it's worth taking a look just
in case.

**Both incidents were fixed in this release.**

### Open pull requests for Homebrew, Krew, Scoop

You can now instead of just pushing to a branch, push and open a pull request.
It even works cross-repository!

Here's an example:

```yaml
# .goreleaser.yml
brews: # can be brews, krew, scoops, etc...
  - # ...
    repository:
      owner: john
      name: repo
      branch: "{{.ProjectName}}-{{.Version}}"
      pull_request:
        enabled: true
        base:
          owner: mike
          name: repo
          branch: main
```

GoReleaser will also read the `.github/PULL_REQUEST_TEMPLATE.md` and prepend it
to the PR description if it exists!

### Nix

We added support to generate Nixpkgs.
We **don't** generate Nixpkgs that compile from source, though.

Instead, we use the already built archives.

This decision was made because this way we can support closed-source software as
well as Open Source.
The idea here is that you create your own [NUR][] and instruct your users to
install from there.

[NUR]: https://github.com/nix-community/NUR

Example:

```yaml
# .goreleaser.yml
nix:
  - name: goreleaser-pro
    repository:
      owner: goreleaser
      name: nur
    homepage: https://goreleaser.com
    description: Deliver Go binaries as fast and easily as possible
    license: unfree
    install: |-
      mkdir -p $out/bin
      cp -vr ./goreleaser $out/bin/goreleaser
      installManPage ./manpages/goreleaser.1.gz
      installShellCompletion ./completions/*
```

### Winget

Now that Winget supports installing `zip` packages, GoReleaser added support to
generate the needed manifests, and you can then PR them to
`microsoft/winget-pkgs`.

![winget-installed goreleaser on windows](https://carlosbecker.com/posts/goreleaser-v1.19/pic.jpg)

Example:

```yaml
# .goreleaser.yml
winget:
  - name: goreleaser-pro
    publisher: goreleaser
    license: Copyright Becker Software LTDA
    copyright: Becker Software LTDA
    homepage: https://goreleaser.com
    short_description: Deliver Go binaries as fast and easily as possible
    repository:
      owner: goreleaser
      name: winget-pkgs
      branch: "goreleaser-pro-{{.Version}}"
      pull_request:
        enabled: true
        draft: true
        base:
          owner: microsoft
          name: winget-pkgs
          branch: master
```

PS: when you open a PR to `microsoft/winget-pkgs`, you are expected to fill the
PR template there... Don't forget to do it! 😄

### Ko improvements

The Ko pipe will now ignore empty tags (e.g. if a template evaluate to an empty
string).

Ko also now properly registers its manifests within GoReleaser's context, so you
can sign them with `docker_signs`.

### Deprecations that were permanently removed

Some things that were deprecated for over 6 months were removed in this release:

- `archives.replacements`
- `archives.rlcp`

There are also other deprecations to be removed soon!

Check the [deprecations][] page to find out more, and run `goreleaser check`
every now and then to see if your configuration file is good!

[deprecations]: https://goreleaser.com/deprecations

### Templates

More fields now accept templates:

- `dockers.skip_push`
- `docker_manifests.skip_push`
- `scoops.description`
- `scoops.homepage`
- `snapcrafts.title`
- `snapcrafts.icon`
- `snapcrafts.assumes`
- `snapcrafts.hooks`

On the same token, there are a couple of new template functions and fields:

- `{{.IsNightly}}` (always false on OSS)
- `{{.Checksums}}` can be used in the release body template
- `{{envOrDefault "FOO" "bar" }}` returns the value of `$FOO` if it is set,
  otherwise returns `bar`

### Standard repository

Historically, you would set `brews.tap`, `krews.index` and etc.
Internally, they all used the same structure: a repository.
"A repository" is also (probably) how most think about these fields.

To make things easier on everyone, now all those fields are named `repository`
instead.

You can check the [deprecations][] page to find more information.

### Continue on error

From this version onward, GoReleaser will not hard-stop when Homebrew, Nix, and
other pipes fail to publish.

Our understanding is that having a broken, stopped-in-the-middle release, is
worse than continuing and reporting all the errors in the end, so you can fix
them all in a single pass and do a point-release.

You can still get the previous behavior by passing the `--fail-fast` flag.

### Upx

As promised, `upx` now has more filters: `goos`, `goarch`, `goarm` and
`goamd64`.

### Telegram

The Telegram announcer now supports choosing the message format.

You can also use `mdv2escape` to escape sequences accordingly to `mdv2`.

### Changelog

Besides just excluding commits that match some regular expressions, you can now
include **only** the commits that match one of them.

Example:

```yaml
# .goreleaser.yml
changelog:
  filters:
    include:
      - "^feat.*"
      - "^fix.*"
```

### Bugfixes et al

We also had a bunch of bugfixes and documentation improvements, as always.

## Other news

- GoReleaser now has ~11.8k stars and 340 contributors! Thanks, everyone!
- We eventually discuss new features in our Discord server. 
  [Join the conversation](https://goreleaser.com/discord)!
- nFPM had new releases as well, 
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can [install][] or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries [here][oss-rel] and
[here (for Pro)][pro-rel].

[install]: https://goreleaser.com/install
[pro-rel]: https://github.com/goreleaser/goreleaser-pro/releases/tag/v1.19.0-pro
[oss-rel]: https://github.com/goreleaser/goreleaser/releases/tag/v1.19.0

## Helping out

You can help by contributing features and bug fixes, or by donating.
You may also be interested in buying a GoReleaser Pro license.

You can find out more [here](https://goreleaser.com/sponsors/).
