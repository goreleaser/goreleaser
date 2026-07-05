---
title: Announcing GoReleaser v2.17
date: 2026-07-04
slug: goreleaser-v2.17
tags: [announcements]
authors: [caarlos0]
---

A packaging and verification release: Windows `.msix` packages, RISC-V 64
packaging through nFPM v2.47, post-release verification, and templated
Dockerfiles in `dockers_v2`.

<!--more-->

## Windows MSIX packages

GoReleaser can now build Windows `.msix` packages through the `nfpms` pipe.

This wires up [nfpm](https://github.com/goreleaser/nfpm)'s MSIX packager, and
started as [@umaidshahid](https://github.com/umaidshahid)'s work in
[#6647](https://github.com/goreleaser/goreleaser/pull/6647) — thanks!

A single `nfpms` entry can list both Linux formats and `msix`: GoReleaser feeds
Windows binaries to the `msix` format and Linux binaries to the others, so each
binary ends up only in the package that matches its platform.

```yaml {filename=".goreleaser.yaml"}
nfpms:
  - formats: [deb, rpm, msix]
    msix:
      publisher: "CN=MyCompany"
      properties:
        logo: ./assets/logo.png
      applications:
        - id: MyApp
          executable: myapp.exe
```

`msix.publisher`, `msix.properties.logo`, and at least one `msix.applications`
entry are required. Unlike the Linux formats, `bindir` doesn't apply — binaries
always land at the root of the package, so each application's `executable` is
simply the binary's file name.

> [!WARNING]
> The `msix` format is experimental.

See the [documentation](https://goreleaser.com/customization/package/nfpm/) for
the full configuration reference, including identity, properties, and signing.

## RISC-V 64 packages, via nFPM v2.47

GoReleaser now bundles
[nFPM v2.47](https://github.com/goreleaser/nfpm/releases/tag/v2.47.0).

The headline for packagers is **RISC-V 64** support: if you build `riscv64`
binaries, GoReleaser can now turn them into `.deb`, `.rpm`, `.apk`, and the
other nFPM formats — no extra configuration needed.

```yaml {filename=".goreleaser.yaml"}
builds:
  - goos: [linux]
    goarch: [riscv64]

nfpms:
  - formats: [deb, rpm, apk]
```

The bump also carries the usual round of dependency and security updates that
flow straight into GoReleaser.

See the [nFPM release notes](https://github.com/goreleaser/nfpm/releases/tag/v2.47.0)
for the full list.

## Soft-float and hard-float ARM builds

Go lets you append an optional floating-point ABI suffix to `GOARM` to control
the assembly emitted for variants with and without an FPU. You can now use that
suffix in your `goarm` targets:

```yaml {filename=".goreleaser.yaml"}
builds:
  - goarm:
      - "6,softfloat"
      - "7,hardfloat"
```

A new `.Abi` template field exposes the selected ABI (`softfloat`/`hardfloat`,
empty unless set). Only one ABI per `GOARM` version is supported, since the two
are indistinguishable once packaged, and `ignore` matches the exact form you
wrote (`goarm: 7` ignores a bare `7` target, not `7,softfloat`).

Thanks to [Marvin Drees](https://github.com/MDr164) for contributing this.

See the [documentation](https://goreleaser.com/customization/builds/builders/go/)
for more details.

## Templated Dockerfiles and build retries in `dockers_v2`

`dockers_v2` gains two options ported from the v1 `dockers` pipe:
`templated_dockerfile` and `templated_extra_files`.

`templated_dockerfile` renders a Dockerfile as a template before building, and
its rendered content is also used to resolve the `.BaseImage` and
`.BaseImageDigest` annotations. `templated_extra_files` renders source files as
templates before copying them into the build context.

Docker builds can also be retried now, which helps with flaky registries and
transient network failures.

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - images:
      - myuser/myimage
    templated_dockerfile: Dockerfile.tmpl
    templated_extra_files:
      - src: config.yml.tmpl
        dst: config.yml
    retry:
      attempts: 5
      delay: 5s
```

See the [documentation](https://goreleaser.com/customization/package/dockers_v2/)
for more details.

## Post-release verification

{{< g_featpro >}}

The new `verify` pipe and `goreleaser verify` command re-download your published
release assets from their public URLs and run your verification commands against
them.

This catches the failures that happen _after_ everything "succeeded": broken or
truncated uploads, bad signatures, and CDN propagation issues.

It is opt-in — add a `verify` section to enable it — and runs automatically at
the end of `goreleaser release` and `goreleaser publish`, right before
announcing. You can skip it with `--skip=verify`, or run it on its own against a
previously prepared `dist` directory with `goreleaser verify`.

```yaml {filename=".goreleaser.yaml"}
verify:
  commands:
    - cmd: sha256sum
      args: ["-c", "{{ .ProjectName }}_{{ .Version }}_checksums.txt"]
```

Commands can run once in the download directory, once per asset, or once per
published image — so verifying blob and image signatures with
[cosign](https://github.com/sigstore/cosign) is a matter of a couple more
commands.

See the [documentation](https://goreleaser.com/customization/verify/) for the
full configuration reference.

## Fallback license keys

{{< g_featpro >}}

The `--key` flag can now be set multiple times. GoReleaser tries each key in
order and uses the first one that validates.

This is mainly useful with offline (air-gapped) license keys: you can provide
your offline key and fall back to the regular online check in case the offline
key is stale — for example, if you renewed your license but forgot to
regenerate the offline key.

```bash
goreleaser release --key goreleaser.key --key "your-online-key"
```

If no `--key` flag is provided, GoReleaser still falls back to the
`GORELEASER_KEY` environment variable, as before.

See the [documentation](https://goreleaser.com/pro/) for more details.

## Other updates

- [**builds**](https://goreleaser.com/customization/builds/): reject empty target strings; return a proper error when `go.mod` is unreadable
- default the pull request branch name for [homebrew_casks](https://goreleaser.com/customization/publish/homebrew_casks/), [scoop](https://goreleaser.com/customization/publish/scoop/), [nix](https://goreleaser.com/customization/publish/nix/), [winget](https://goreleaser.com/customization/publish/winget/), and [krew](https://goreleaser.com/customization/publish/krew/)
- skip the GitHub `merge-upstream` sync when the target repo isn't a fork
- [**scm**](https://goreleaser.com/customization/publish/scm/): allow a custom token on the release repository
- [**winget**](https://goreleaser.com/customization/publish/winget/): configure the manifest locale via `default_locale` (defaults to `en-US`)
- [**nfpm**](https://goreleaser.com/customization/package/nfpm/): produce a valid arch for Termux packages
- [**mcp**](https://goreleaser.com/customization/publish/mcp/): clean the subfolder path
- **dependencies**: Go 1.26.4; `golang.org/x/net` and `go-pkcs12` security updates; dropped the `docker/docker` dependency
- [**pro/templates**](https://goreleaser.com/customization/general/templates/): new `.Dist` template variable with the absolute path to the `dist` directory
- [**pro/dockers_v2**](https://goreleaser.com/customization/package/dockers_v2/): `hooks` now honor the `if` field
- [**pro/npm**](https://goreleaser.com/customization/publish/npm/): use a universal archive for darwin `arm64` and `amd64`

## Other news

- GoReleaser now has ~15.9k stars and 474 contributors! Thanks, everyone!
- You can follow release updates on our
  [Telegram channel](https://t.me/goreleasernews).
- We've decided to [close our Discord server](https://goreleaser.com/blog/closing-discord/) —
  please use [GitHub Discussions](https://github.com/goreleaser/goreleaser/discussions)
  instead.
- nFPM had new releases as well,
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< g_button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.17.0" label="v2.17.0 (OSS)" icon="github" primary="false" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.17.0" label="v2.17.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< g_button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< g_button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}
