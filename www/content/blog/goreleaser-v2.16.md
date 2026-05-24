---
title: Announcing GoReleaser v2.16
date: 2026-05-24
slug: goreleaser-v2.16
tags: [announcements]
authors: [caarlos0]
---

Immutable releases, a new build target for Node.js, `dockers_v2` graduating
out of experimental, and the legacy `brews` config finally getting the boot.

<!--more-->

## Immutable releases

`v2.16.0` is the first non-nightly GoReleaser release published under our
new [immutable releases policy](https://goreleaser.com/blog/immutable-releases/).
Once a tag is published, its bytes can never be replaced — pinning to
`v2.16.0` gives you the exact same artifacts forever.

Nightlies changed too: instead of overwriting a single moving `nightly` tag,
each nightly run now creates its own immutable tag (e.g.
`v2.16.0-abc1234-nightly`). The old `nightly` tag is frozen and will be
deleted soon — if you're still pulling it, switch to the new format. To keep
the release list from growing forever, old nightly tags are periodically
pruned.

If you use [`goreleaser-action`](https://github.com/goreleaser/goreleaser-action)
(≥ v7.2.0), the new nightly tag format is handled for you — no config
changes needed.

See the blog post for the background (supply-chain safety) and the full
nightly transition details.

## Node.js single-executable apps

GoReleaser can now build [Node.js Single Executable Application][sea] (SEA)
binaries via the new `node` builder!

This started as [@vedantmgoyal9](https://github.com/vedantmgoyal9)'s work in
[#6136](https://github.com/goreleaser/goreleaser/pull/6136) — thanks for
getting the ball rolling.

A minimal example:

```yaml {filename=".goreleaser.yaml"}
builds:
  - builder: node
    main: index.js
    targets:
      - linux-x64
      - darwin-arm64
      - win-x64
```

GoReleaser fetches the right Node.js distribution for each target and bundles
your entrypoint into a single executable. By default it builds for
`darwin-arm64`, `darwin-x64`, `linux-arm64`, `linux-x64`, `win-arm64`, and
`win-x64`.

> [!WARNING]
> The builder is experimental and requires Node ≥ v25.5.0.

See the [documentation](https://goreleaser.com/customization/builds/builders/node/)
for the full configuration reference, including custom Node binaries, hooks
to bundle dependencies, and per-target environment variables. There's also a
working [example repository](https://github.com/goreleaser/example-node) to
get you started.

[sea]: https://nodejs.org/api/single-executable-applications.html

## Docker (v2) is no longer experimental

`dockers_v2` is no longer marked as experimental — the experimental warning
is gone, and the pipe is ready for production use.

It also picks up three improvements this release.

The Dockerfile is now parsed to expose `.BaseImage` and `.BaseImageDigest`
template variables, which makes it trivial to set the standard OCI base-image
labels:

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - images:
      - myuser/myimage
    labels:
      "org.opencontainers.image.base.name": "{{ .BaseImage }}"
      "org.opencontainers.image.base.digest": "{{ .BaseImageDigest }}"
```

The resulting artifact now carries the list of built platforms in
`extra.platforms`, which custom publishers and other pipes can use to route
the image correctly.

And finally, you can now run `pre` and `post` hooks around the actual
`docker buildx build` invocation — handy for staging files into the build
context, or for tagging/scanning the resulting image:

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - images:
      - myuser/myimage
    hooks:
      pre:
        - cmd: ./scripts/before-docker.sh
          dir: "{{ .ContextDir }}"
      post:
        - cmd: ./scripts/after-docker.sh {{ .Digest }}
```

Hooks get the resolved `.Dockerfile`, `.Images`, and `.ContextDir`, plus the
final `.Digest` on `post` hooks.

See the [documentation](https://goreleaser.com/customization/package/dockers_v2/)
for more details.

## `brews` is now fully deprecated

The legacy `brews` config — which generated _hackyish_ Homebrew formulas that
installed pre-compiled binaries — has been soft-deprecated since v2.10 and is
now fully deprecated.

Migrate to [`homebrew_casks`](https://goreleaser.com/customization/publish/homebrew_casks/),
which is the right tool for the job: it's how Homebrew expects pre-compiled
binaries to be distributed, and it gets all the new features (completion
generation, post-install hooks, and so on).

See the [deprecation notice](https://goreleaser.com/resources/deprecations/#brews)
for the migration guide.

## A note on the v2.15.x patch series

If you skipped any of the v2.15.x patches, **v2.15.3** in particular is worth
calling out: it shipped a security fix that prevents secret leaks in logs and
improves redaction, along with a large pass of panic guards and retry
improvements across most pipes (Docker, GitHub, gomod, nfpm, Rust, SBOM,
templates, and more).

If you're on v2.15.0–v2.15.2, please upgrade.

## Other updates

- [**archives**](https://goreleaser.com/customization/package/archives/): added `xz` format (thanks to [Jared Allard](https://github.com/jaredallard))
- [**nightly**](https://goreleaser.com/customization/publish/nightlies/): support templates in `tag_name`; defer tag templating to evaluation time; `run` scripts now resolve nightly tags correctly
- [**nix**](https://goreleaser.com/customization/publish/nix/): support `meta.mainProgram`
- [**release**](https://goreleaser.com/customization/publish/scm/): preserve prerelease state on publish; handle GitHub secondary rate limits; remove author lookup by email
- [**homebrew_casks**](https://goreleaser.com/customization/publish/homebrew_casks/): emit `generate_completions_from_executable` after `postflight`
- [**srpm**](https://goreleaser.com/customization/package/srpm/): set `format` and `extension` on the artifact
- [**webhook**](https://goreleaser.com/customization/announce/webhook/) and [**linkedin**](https://goreleaser.com/customization/announce/linkedin/): error-handling improvements
- **dependencies**: Go 1.26.3
- [**pro/cloudsmith**](https://goreleaser.com/customization/publish/cloudsmith/): upload Source RPMs; clearer error messages
- [**pro/gemfury**](https://goreleaser.com/customization/publish/gemfury/): upload Source RPMs; include response body in errors
- [**pro/templates**](https://goreleaser.com/customization/general/templates/): new `listExclude` template function
- [**pro/sign**](https://goreleaser.com/customization/sign/): allow signing macOS `.pkg` files
- [**pro/nightly**](https://goreleaser.com/customization/publish/nightlies/): keep `.Tag` populated; trim `v` prefix in `version_template`; preserve prerelease flag
- [**pro/npm**](https://goreleaser.com/customization/publish/npm/): omit empty `engines` from `package.json`

## Other news

- GoReleaser now has ~15.8k stars and 432 contributors! Thanks, everyone!
- You can follow release updates on our
  [Telegram channel](https://t.me/goreleasernews).
- We often discuss new features in our Discord server.
  [Join the conversation][discord]!
- nFPM had new releases as well,
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< g_button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.16.0" label="v2.16.0 (OSS)" icon="github" primary="false" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.16.0" label="v2.16.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< g_button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< g_button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}

[discord]: https://goreleaser.com/discord
