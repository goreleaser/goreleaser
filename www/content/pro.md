---
title: "GoReleaser Pro"
breadcrumbs: false
weight: 400
sidebar:
  hide: true
---

GoReleaser Pro is a paid, closed-source GoReleaser distribution with some
additional features:

- Create [macOS installers (`.pkg`)](/customization/package/pkg/);
- Create [Windows installers (`.exe`) with NSIS](/customization/package/nsis/);
- Smart [SemVer tag sorting](/customization/general/git/#semver-sorting);
- Publish to [NPM registries](/customization/publish/npm/);
- [Native sign and notarize](/customization/sign/notarize/#native)
  macOS App Bundles, Disk Images, and Installers;
- Use [AI](/customization/publish/changelog/#enhance-with-ai) to improve/format
  your release notes;
- Further filter artifacts with `if` statements;
- Create [macOS App Bundles (`.app`)](/customization/package/app_bundles/);
- Easily create `alpine`, `apt`, and `yum` repositories with the
  [CloudSmith integration](/customization/publish/cloudsmith/);
- Have [global defaults for homepage, description, etc](/customization/general/metadata/);
- Run [hooks before publishing](/customization/publish/beforepublish/) artifacts;
- Cross publish (e.g. releases to GitLab, pushes Homebrew Tap to GitHub);
- Publish [versioned Homebrew Casks](/customization/publish/homebrew_casks/#versioned-casks);
- Keep [DockerHub image descriptions up to date](/customization/publish/dockerhub/);
- Create [macOS disk images (`.dmg`)](/customization/package/dmg/);
- Create [Windows installers (`.msi`) with Wix](/customization/package/msi/);
- Use `goreleaser release --single-target` to build the whole pipeline for a
  single architecture locally;
- Check boxes in pull request templates;
- [Template entire files](/customization/general/templatefiles/) and add them to the
  release. You can also template files that will be included in archives,
  packages, Docker images, etc...;
- Use the [`.Artifacts`](/customization/general/templates/#artifacts) template
  variable to build more powerful customizations;
- [Split and merge builds](/customization/general/partial/) to speed up your release
  by splitting work, use CGO, or run platform-specific code;
- More [changelog options](/customization/publish/changelog/): Filter commits by path
  & subgroups, group dividers;
- Have custom [before and after hooks for archives](/customization/package/archives/);
- Prepare a release with
  `goreleaser release --prepare`,
  publish and announce it later with
  `goreleaser publish` and
  `goreleaser announce`, or with
  `goreleaser continue`;
- Preview and test your next release's change log with
  `goreleaser changelog`;
- Continuously release [nightly builds](/customization/publish/nightlies/);
- Import pre-built binaries with the
  [`prebuilt` builder](/customization/builds/builders/prebuilt/);
- Rootless build [Docker images](/customization/package/docker/#using-podman)
  and
  [manifests](/customization/package/docker_manifest/#using-podman) with
  [Podman](https://podman.io);
- Easily create `apt`, `yum`, and alpine repositories with the
  [gemfury.io integration](/customization/publish/gemfury/);
- Reuse configuration files with the
  [include keyword](/customization/general/includes/);
- Run commands after the release with
  [global after hooks](/customization/general/hooks/);
- Use GoReleaser within your [monorepo](/customization/monorepo/);
- Create
  [custom template variables](/customization/general/templates/#custom-variables)
  (goes well with [includes](/customization/general/includes/)).

## Pricing

{{< tabs >}}

{{< tab "Yearly" >}}

{{< cards cols="2" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=5Ut2O2lkGo6A2yu9q9HXyg%3D%3D&recurrence=yearly&wanted=true" title="Personal — $165/yr" subtitle="For your personal projects." icon="user" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=e1n1yQCH968rENm8w0FBgQ%3D%3D&recurrence=yearly&wanted=true" title="Startup — $247/yr" subtitle="For small companies (up to 20 people) and up to 5 repositories." icon="user-group" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=0XPP8t1FY9Y--XPBkS6PlQ%3D%3D&recurrence=yearly&wanted=true" title="Business — $948/yr" subtitle="For big companies (up to 100 people) and up to 20 repositories. Can create and use air-gapped licenses." icon="briefcase" tag="most popular" tagColor="green" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=RKh9SHYqKo8Nx_MWpIWj0g%3D%3D&recurrence=yearly&wanted=true" title="Enterprise — $3,300/yr" subtitle="For bigger companies. Unlimited users and repositories. Can create and use air-gapped licenses." icon="office-building" >}}
{{< /cards >}}

{{< /tab >}}

{{< tab "Monthly" >}}

{{< cards cols="2" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=5Ut2O2lkGo6A2yu9q9HXyg%3D%3D&recurrence=monthly&wanted=true" title="Personal — $15/mo" subtitle="For your personal projects." icon="user" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=e1n1yQCH968rENm8w0FBgQ%3D%3D&recurrence=monthly&wanted=true" title="Startup — $22/mo" subtitle="For small companies (up to 20 people) and up to 5 repositories." icon="user-group" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=0XPP8t1FY9Y--XPBkS6PlQ%3D%3D&recurrence=monthly&wanted=true" title="Business — $86/mo" subtitle="For big companies (up to 100 people) and up to 20 repositories. Can create and use air-gapped licenses." icon="briefcase" tag="most popular" tagColor="green" >}}
{{< card link="https://gumroad.com/l/CadfZ?option=RKh9SHYqKo8Nx_MWpIWj0g%3D%3D&recurrence=monthly&wanted=true" title="Enterprise — $300/mo" subtitle="For bigger companies. Unlimited users and repositories. Can create and use air-gapped licenses." icon="office-building" >}}
{{< /cards >}}

{{< /tab >}}

{{< /tabs >}}

> [!TIP]
> Save up to **8%** with yearly billing.
> All prices are in USD.

## Using GoReleaser Pro

GoReleaser Pro is a different binary, see the [install options](/getting-started/install).
Once you have it, you can use the serial key with either `--key` or by setting
`GORELEASER_KEY`.

See [this page](/post-checkout/) for more information.

Once you [buy it](https://gum.co/goreleaser), you'll get a license key. You can
then pass it to the `release` command either via the
`--key` flag or the `GORELEASER_KEY` environment variable.

If you use the GitHub action, you will want to set the `distribution` option to
`goreleaser-pro`. Check the [documentation](/customization/ci/actions/) for more details.

### Offline licenses

{{< g_version "v2.14" >}}

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

> [!WARNING]
> Offline licenses expire based on your billing cycle (at most 90 days).
> You will need to re-export before expiry.

> [!NOTE]
> Offline licenses are available on the **Business** and **Enterprise** plans.

## Road map

We don't have a properly organized public road map, but we are always open to
suggestions!

Once you subscribe, feel free to
[email me](mailto:carlos@becker.software?subject=GoReleaser%20Feature%20Suggestion)
with your suggestions and ideas.

## Sponsors & discounts

- Prices may increase as we keep adding more pro-only features, so lock in your
  rate now;
- If you sponsor either the project or any of its developers, you [can ask for a
  discount](mailto:carlos@becker.software?subject=GoReleaser%20Coupon%20Request)!

## Enterprise support

We can also provide enterprise support contracts, with proper SLAs and help
designing and updating your release pipelines.
If this sound interesting to you, feel free to
[contact us](mailto:carlos@becker.software?subject=GoReleaser%20Enterprise%20Support).

## EULA

Please, make sure you read and agree with our [EULA](/resources/eula/).

---

**✨✨ Thanks for your support! ✨✨**
