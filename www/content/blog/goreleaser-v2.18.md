---
title: Announcing GoReleaser v2.18
date: 2026-08-23
slug: goreleaser-v2.18
tags: [announcements]
authors: [caarlos0]
---

An observability release: a release summary in GitHub Actions, preflight checks
that fail before you build, OpenTelemetry traces, a new Iru Custom Apps
publisher, and Go 1.27.

<!--more-->

## Release summary in GitHub Actions

When GoReleaser runs inside [GitHub Actions](https://goreleaser.com/customization/ci/actions/),
it now writes a summary of what your release published to the job summary view,
so you can see the outcome of a run without reading its logs.

There is nothing to configure. GoReleaser appends to the file named by the
`GITHUB_STEP_SUMMARY` environment variable, which GitHub Actions sets for every
step.

Each notable action becomes one bullet, with a link when there is one:

```text
- Published v1.2.3 to GitHub with 12 assets: https://github.com/user/repo/releases/tag/v1.2.3
- Pushed Docker image `user/foo:v1.2.3`
- Updated homebrew formula `foo.rb` in `user/homebrew-tap`
- Opened pull request to `user/nur` (nixpkg `foo.nix`): https://github.com/user/nur/pull/12
- Pushed `foo` to npm
- Uploaded 8 files to `s3://my-bucket`
```

An action is only reported after it succeeds, so the summary describes what
actually happened, not what was configured. Releases, packages, images,
manifests, uploads, and announcements are all covered.

See the [documentation](https://goreleaser.com/customization/summary/) for the
full list of what gets reported.

## Preflight checks

Some releases fail at the very last step, after you already waited for every
build, package, and signature — because the token can't publish, or because the
tag is already out there as an immutable release that can't be updated.

GoReleaser now checks for that up front:

```yaml {filename=".goreleaser.yaml"}
release:
  preflight:
    fail_on_error: true
```

With `fail_on_error` set, a failed check aborts the release before building.
When it's false (the default), a failed check only logs a warning, so nothing
changes for existing configurations until you opt in.

See the [documentation](https://goreleaser.com/customization/publish/scm/) for
more details.

## Iru Custom Apps

GoReleaser can now publish artifacts as _Custom Apps_ to
[iru.com](https://www.iru.com) (formerly Kandji) endpoint management, making
them available for deployment to your managed Macs.

```yaml {filename=".goreleaser.yaml"}
iru:
  url: https://mycompany.api.kandji.io
  name: "My App {{ .Version }}"
  install_type: package
```

Custom Apps are a macOS-only library item: the supported install types
(`package`, `zip`, `image`) correspond to `.pkg`, `.zip`, and `.dmg` files, and
the pre/post install scripts run on the target Mac. GoReleaser can create a new
Custom App on every release, or update an existing one through
`library_item_id`.

Thanks to [Wim Wenigerkind](https://github.com/wimwenigerkind) for contributing
this.

See the [documentation](https://goreleaser.com/customization/publish/iru/) for
the full configuration reference.

## winget locale manifests

`winget` entries can now publish additional locale manifests alongside the
default one:

```yaml {filename=".goreleaser.yaml"}
winget:
  - name: foo
    publisher: myself
    short_description: "My amazing project"
    license: MIT
    repository:
      owner: myself
      name: winget-pkgs
    additional_locales:
      - locale: pt-BR
        short_description: "Meu projeto incrível"
      - locale: fr-FR
        short_description: "Mon projet incroyable"
```

Every field the default locale accepts can be overridden per locale, and
anything you leave out falls back to the default locale's value.

Thanks to [Mohammed Anasuddin Zaid](https://github.com/MohammedAnasuddinZaid)
for contributing this.

See the [documentation](https://goreleaser.com/customization/publish/winget/)
for more details.

## OpenTelemetry traces

{{< g_featenterprise >}}

GoReleaser can now export [OpenTelemetry](https://opentelemetry.io) traces of
your releases, so you can see where the time goes: which pipes are slow, which
builds dominate, which upload is flaky, and how it all trends over time.

```yaml {filename=".goreleaser.yaml"}
telemetry:
  enabled: true
```

Traces are always sent to **your** collector, configured through the standard
`OTEL_*` environment variables. GoReleaser never sends anything to us.

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="https://otel.mycompany.com"
export OTEL_EXPORTER_OTLP_HEADERS="authorization=******"
goreleaser release
```

It is opt-in on purpose: without the configuration option, GoReleaser would
start talking to the network whenever a CI runner happens to have `OTEL_*`
variables set for other tooling. You have to ask for it.

See the [documentation](https://goreleaser.com/customization/telemetry/) for the
full reference.

## Other updates

- **dependencies**: Go 1.27
- [**scm**](https://goreleaser.com/customization/publish/scm/): pull request creation can use a different auth token, thanks to [Emily Curry](https://github.com/emily-curry)
- [**ko**](https://goreleaser.com/customization/package/ko/): `local_domain`, `base_image`, and `repositories` support templating, thanks to [Manuel Rüger](https://github.com/mrueg)
- [**winget**](https://goreleaser.com/customization/publish/winget/): count `arm64` in the duplicate-archive check; fall back to the default description in additional locales
- [**builds**](https://goreleaser.com/customization/builds/): node windows targets get their `.exe` extension back
- [**nfpm**](https://goreleaser.com/customization/package/nfpm/): overrides no longer ignore `package_name`, `epoch`, `release`, and `prerelease`; record the conventional extension instead of the format name; don't set the deb arch variant for `goamd64` v1
- [**srpm**](https://goreleaser.com/customization/package/srpm/): the documented RPM fields are actually configurable now
- [**homebrew**](https://goreleaser.com/customization/publish/homebrew_casks/): emit casks that pass `brew style`; a formula or cask without a repository no longer stops the ones after it
- a disabled or skipped [flatpak](https://goreleaser.com/customization/package/flatpak/), [snap](https://goreleaser.com/customization/package/snapcraft/), [nix](https://goreleaser.com/customization/publish/nix/), [winget](https://goreleaser.com/customization/publish/winget/), [krew](https://goreleaser.com/customization/publish/krew/), [milestone](https://goreleaser.com/customization/publish/milestone/), or [upload](https://goreleaser.com/customization/publish/upload/) entry no longer stops the ones after it
- [**snapcraft**](https://goreleaser.com/customization/package/snapcraft/): `assumes`, `hooks`, and `plugs` are no longer dropped when `apps` is omitted
- [**blob**](https://goreleaser.com/customization/publish/blob/): honor `s3_force_path_style` without a custom endpoint
- [**aur**](https://goreleaser.com/customization/publish/aur/): expand description templates before escaping quotes
- [**chocolatey**](https://goreleaser.com/customization/package/chocolatey/): error on multiple archives for the same platform
- [**changelog**](https://goreleaser.com/customization/publish/changelog/): a commit matching two include filters is no longer listed twice
- [**sign**](https://goreleaser.com/customization/sign/): `artifacts: none` no longer masks real signing failures
- [**notarize**](https://goreleaser.com/customization/sign/notarize/): cap the macOS notarization timeout at 20m
- [**templates**](https://goreleaser.com/customization/general/templates/): register the documented `join` function; tag templates no longer strip apostrophes from the message
- [**mcp**](https://goreleaser.com/customization/publish/mcp/): `mcp.disable` is read now, as documented
- [**docker**](https://goreleaser.com/customization/package/docker/): `gpg-agent` is included in the image; fixed `dockers_v2` annotation scopes
- **security**: `go-openapi/spec` and `go-openapi/validate` updates

## Other news

- GoReleaser now has ~16.0k stars and 495 contributors! Thanks, everyone!
- You can follow release updates on our
  [Telegram channel](https://t.me/goreleasernews).
- nFPM had new releases as well,
  [check it out](https://github.com/goreleaser/nfpm/releases).

## Download

You can install or upgrade using your favorite package manager, or see the
full release notes and download the pre-compiled binaries from GitHub:

{{< g_button href="https://goreleaser.com/install" label="Install" icon="download" primary="true" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser/releases/tag/v2.18.0" label="v2.18.0 (OSS)" icon="github" primary="false" >}}
{{< g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/v2.18.0" label="v2.18.0 (Pro)" icon="github" primary="false" >}}

## Helping out

You can help by reporting issues, contributing features, documentation
improvements, and bug fixes.
You can also sponsor the project, or get a GoReleaser Pro license.

{{< g_button href="https://goreleaser.com/pro" label="Get the Pro license" icon="pro" primary="true" >}}
{{< g_button href="https://goreleaser.com/sponsors" label="Sponsor the project" icon="sponsor" primary="false" >}}
