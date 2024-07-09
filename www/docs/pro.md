# GoReleaser Pro

GoReleaser Pro is a paid, closed-source GoReleaser distribution with some
additional features:

- [x] Easily create `alpine`, `apt`, and `yum` repositories with the
      [CloudSmith integration](customization/cloudsmith.md);
- [x] Have [global defaults for homepage, description, etc](customization/metadata.md);
- [x] Run [hooks before publishing](customization/beforepublish.md) artifacts;
- [x] Cross publish (e.g. releases to GitLab, pushes Homebrew Tap to GitHub);
- [x] Keep [DockerHub image descriptions up to date](customization/dockerhub.md);
- [x] Create [macOS disk images (DMGs)](customization/dmg.md);
- [x] Create [Windows installers](customization/msi.md);
- [x] Use `goreleaser release --single-target` to build the whole pipeline for a
      single architecture locally;
- [x] Check boxes in pull request templates;
- [x] [Template entire files](customization/templatefiles.md) and add them to the
      release. You can also template files that will be included in archives,
      packages, Docker images, etc...;
- [x] Use the [`.Artifacts`](customization/templates.md/#artifacts) template
      variable to build more powerful customizations;
- [x] [Split and merge builds](customization/partial.md) to speed up your release
      by splitting work, use CGO, or run platform-specific code;
- [x] More [changelog options](customization/changelog.md): Filter commits by path
      & subgroups, group dividers;
- [x] Have custom [before and after hooks for archives](customization/archive.md);
- [x] Prepare a release with
      [`goreleaser release --prepare`](cmd/goreleaser_release.md), publish and
      announce it later with
      [`goreleaser publish`](cmd/goreleaser_publish.md) and
      [`goreleaser announce`](cmd/goreleaser_announce.md), or with
      [`goreleaser continue`](cmd/goreleaser_continue.md);
- [x] Preview and test your next release's change log with
      [`goreleaser changelog`](cmd/goreleaser_changelog.md);
- [x] Continuously release [nightly builds](customization/nightlies.md);
- [x] Import pre-built binaries with the
      [`prebuilt` builder](customization/builds.md#import-pre-built-binaries);
- [x] Rootless build [Docker images](customization/docker.md#using-podman) and
      [manifests](customization/docker_manifest.md#using-podman) with
      [Podman](https://podman.io);
- [x] Easily create `apt` and `yum` repositories with the
      [fury.io integration](customization/fury.md);
- [x] Reuse configuration files with the
      [include keyword](customization/includes.md);
- [x] Run commands after the release with
      [global after hooks](customization/hooks.md);
- [x] Use GoReleaser within your [monorepo](customization/monorepo.md);
- [x] Create
      [custom template variables](customization/templates.md#custom-variables)
      (goes well with [includes](customization/includes.md)).

<script src="https://gumroad.com/js/gumroad.js"></script>

<a class="gumroad-button" href="https://gumroad.com/l/CadfZ" target="_blank">Get GoReleaser Pro</a>

## Road map

We don't have a properly organized public road map (_yet_), but these are some
of the things we plan to work on, in one form or another:

- [ ] `--dry-run` to test the release locally, possibly skipping the actual
      build of the binaries to focus on faster iteration of the other parts;

That said, your input is always welcome!
Once you buy it, feel free to
[email me](mailto:carlos@becker.software?subject=GoReleaser%20Feature%20Suggestion)
with your suggestions and ideas.

## Pricing & Sponsors

- The current pricing is low and is likely to increase as we keep adding more
  pro-only features;
- If you sponsor either the project or any of its developers, you [can ask for a
  discount](mailto:carlos@becker.software?subject=GoReleaser%20Coupon%20Request)!

## Enterprise support

I don't have a plan for that yet, but please [email
me](mailto:carlos@becker.software?subject=GoReleaser%20Enterprise%20Support) if
you are interested.

## Using GoReleaser Pro

Once you [buy it](https://gum.co/goreleaser), you'll get a license key. You can
then pass it to the [`release` command](cmd/goreleaser_release.md) either via the
`--key` flag or the `GORELEASER_KEY` environment variable.

If you use the GitHub action, you will want to set the `distribution` option to
`goreleaser-pro`. Check the [documentation](ci/actions.md) for more details.

## EULA

Please, make sure you read and agree with our [EULA](eula.md).

---

**✨✨ Thanks for your support! ✨✨**
