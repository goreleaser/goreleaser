# GoReleaser Pro

GoReleaser Pro is a paid, closed-source GoReleaser distribution with some
additional features:

- [x] [Split and merge builds](/customization/partial) to speed up your release
  by splitting work, use CGO, or run platform-specific code;
- [x] More [changelog options](/customization/changelog): Filter commits by path
  & sub-groups;
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
  [`prebuilt` builder](/customization/build/#import-pre-built-binaries);
- [x] Rootless build [Docker images](/customization/docker/#podman) and
  [manifests](/customization/docker_manifest/#podman) with
  [Podman](https://podman.io);
- [x] Easily create `apt` and `yum` repositories with the
  [fury.io integration](/customization/fury/);
- [x] Reuse configuration files with the
  [include keyword](/customization/includes/);
- [x] Run commands after the release with
  [global after hooks](/customization/hooks/);
- [x] Use GoReleaser within your [monorepo](/customization/monorepo/);
- [x] Create
  [custom template variables](/customization/templates/#custom-variables)
  (goes well with [includes](/customization/includes/)).

<script src="https://gumroad.com/js/gumroad.js"></script>
<a class="gumroad-button" href="https://gumroad.com/l/CadfZ" target="_blank">Get GoReleaser Pro</a>

## Road map

We don't have a properly organized public road map (*yet*), but these are some
of the things we plan to work on, in one form or another:

- [ ] `--dry-run` to test the release locally, possibly skipping the actual
  build of the binaries to focus on faster iteration of the other parts;
- [ ] `--single-target` & friends for `goreleaser release`;
- [ ] first-class macOS signing;
- [ ] create Windows installers;

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
then pass it to the [`release` command](/cmd/goreleaser_release/) either via the
`--key` flag or the `GORELEASER_KEY` environment variable.

If you use the GitHub action, you will want to set the `distribution` option to
`goreleaser-pro`. Check the [documentation](/ci/actions/) for more details.

## EULA

Please, make sure you read and agree with our [EULA](/eula).

---

**✨✨ Thanks for your support! ✨✨**
