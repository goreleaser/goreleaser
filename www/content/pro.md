---
title: "GoReleaser Pro"
weight: 400
sidebar:
  hide: true
---

GoReleaser Pro is a paid, closed-source GoReleaser distribution with some
additional features:

- [x] Create [macOS installers (`.pkg`)](/customization/pkg/);
- [x] Create [Windows installers (`.exe`) with NSIS](/customization/nsis/);
- [x] Smart [SemVer tag sorting](/customization/git/#semver-sorting);
- [x] Publish to [NPM registries](/customization/npm/);
- [x] [Native sign and notarize](/customization/notarize/#native)
      macOS App Bundles, Disk Images, and Installers;
- [x] Use [AI](/customization/changelog/#enhance-with-ai) to improve/format
      your release notes;
- [x] Further filter artifacts with `if` statements;
- [x] Create [macOS App Bundles (`.app`)](/customization/app_bundles/);
- [x] Easily create `alpine`, `apt`, and `yum` repositories with the
      [CloudSmith integration](/customization/cloudsmith/);
- [x] Have [global defaults for homepage, description, etc](/customization/metadata/);
- [x] Run [hooks before publishing](/customization/beforepublish/) artifacts;
- [x] Cross publish (e.g. releases to GitLab, pushes Homebrew Tap to GitHub);
- [x] Keep [DockerHub image descriptions up to date](/customization/dockerhub/);
- [x] Create [macOS disk images (`.dmg`)](/customization/dmg/);
- [x] Create [Windows installers (`.msi`) with Wix](/customization/msi/);
- [x] Use `goreleaser release --single-target` to build the whole pipeline for a
      single architecture locally;
- [x] Check boxes in pull request templates;
- [x] [Template entire files](/customization/templatefiles/) and add them to the
      release. You can also template files that will be included in archives,
      packages, Docker images, etc...;
- [x] Use the [`.Artifacts`](/customization/templates.md/#artifacts) template
      variable to build more powerful customizations;
- [x] [Split and merge builds](/customization/partial/) to speed up your release
      by splitting work, use CGO, or run platform-specific code;
- [x] More [changelog options](/customization/changelog/): Filter commits by path
      & subgroups, group dividers;
- [x] Have custom [before and after hooks for archives](/customization/archive/);
- [x] Prepare a release with
      [`goreleaser release --prepare`](/cmd/goreleaser_release/), publish and
      announce it later with
      [`goreleaser publish`](/cmd/goreleaser_publish/) and
      [`goreleaser announce`](/cmd/goreleaser_announce/), or with
      [`goreleaser continue`](/cmd/goreleaser_continue/);
- [x] Preview and test your next release's change log with
      [`goreleaser changelog`](/cmd/goreleaser_changelog/);
- [x] Continuously release [nightly builds](/customization/nightlies/);
- [x] Import pre-built binaries with the
      [`prebuilt` builder](/customization/prebuilt/);
- [x] Rootless build [Docker images](/customization/docker/#using-podman) and
      [manifests](/customization/docker_manifest/#using-podman) with
      [Podman](https://podman.io);
- [x] Easily create `apt`, `yum`, and alpine repositories with the
      [gemfury.io integration](/customization/gemfury/);
- [x] Reuse configuration files with the
      [include keyword](/customization/includes/);
- [x] Run commands after the release with
      [global after hooks](/customization/hooks/);
- [x] Use GoReleaser within your [monorepo](/customization/monorepo/);
- [x] Create
      [custom template variables](/customization/templates/#custom-variables)
      (goes well with [includes](/customization/includes/)).

<script src="https://gumroad.com/js/gumroad.js"></script>

<a class="gumroad-button" href="https://gumroad.com/l/CadfZ" target="_blank">Get
GoReleaser Pro</a>

## Using GoReleaser Pro

GoReleaser Pro is a different binary, see the [install options](/getting-started/install/#pro).
Once you have it, you can use the serial key with either `--key` or by setting
`GORELEASER_KEY`.

See [this page](/post-checkout/) for more information.

Once you [buy it](https://gum.co/goreleaser), you'll get a license key. You can
then pass it to the [`release` command](/cmd/goreleaser_release/) either via the
`--key` flag or the `GORELEASER_KEY` environment variable.

If you use the GitHub action, you will want to set the `distribution` option to
`goreleaser-pro`. Check the [documentation](/ci/actions/) for more details.

### Offline licenses

{{< version "v2.14" >}}

If you run GoReleaser in an environment without internet access (air-gapped),
you can export an offline license and use it instead of the regular key.

Offline licenses are verified locally — no network calls are made
during verification.

**Exporting an offline license:**

```bash
goreleaser license-export --key "your-license-key" -o goreleaser.key
```

This contacts the GoReleaser signing server, verifies your subscription, and
writes a signed license blob to the given file.
You can also export to `STDOUT` with `-o -`.

**Using an offline license:**

Set `GORELEASER_KEY` to the contents of the exported file:

```bash
export GORELEASER_KEY="$(cat goreleaser.key)"
goreleaser release
```

GoReleaser will automatically detect the offline license and verify it locally.

{{< callout type="warning" >}}

Offline licenses expire based on your billing cycle (at most 90 days).
You will need to re-export before expiry.
{{< /callout >}}

{{< callout type="info" >}}

Offline licenses are available on the **Business** and **Enterprise** plans.
{{< /callout >}}

## Road map

We don't have a properly organized public road map, but we are always open to
suggestions!

Once you subscribe, feel free to
[email me](mailto:carlos@becker.software?subject=GoReleaser%20Feature%20Suggestion)
with your suggestions and ideas.

## Pricing & Sponsors

- The current pricing is low and is likely to increase as we keep adding more
  pro-only features;
- If you sponsor either the project or any of its developers, you [can ask for a
  discount](mailto:carlos@becker.software?subject=GoReleaser%20Coupon%20Request)!

## Enterprise support

We can also provide enterprise support contracts, with proper SLAs and help
designing and updating your release pipelines.
If this sound interesting to you, feel free to
[contact us](mailto:carlos@becker.software?subject=GoReleaser%20Enterprise%20Support).

## EULA

Please, make sure you read and agree with our [EULA](/eula/).

---

**✨✨ Thanks for your support! ✨✨**
