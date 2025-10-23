# Nightlies

<!-- md:pro -->

Whether you need beta builds or a rolling-release system, the nightly builds
feature will do it for you.

To enable it, you must use the `--nightly` flag in the
[`goreleaser release` command](../cmd/goreleaser_release.md).

You also have some customization options available:

```yaml title=".goreleaser.yaml"
nightly:
  # Allows you to change the version of the generated nightly release.
  #
  # Note that some pipes require this to be semantic version compliant (nfpm,
  # for example).
  #
  # Default: `{{ incpatch .Version }}-{{ .ShortCommit }}-nightly`.
  # Templates: allowed.
  version_template: "{{ incpatch .Version }}-devel"

  # Tag name to create if publish_release is enabled.
  tag_name: devel

  # Whether to publish a release or not.
  # Only works on GitHub.
  publish_release: true

  # Whether to delete previous pre-releases for the same `tag_name` when
  # releasing.
  # This allows you to keep a single pre-release.
  keep_single_release: true

  # Whether to publish the nightly as a draft release.
  # Notice that this might not play well with 'keep_single_release', as you'll
  # end up with no published nightly releases.
  #
  # Default: value of 'release.draft'.
  # <!-- md:inline_version v2.12 -->.
  draft: true
```

!!! warning

    Do not use the `{{.Tag}}` field in the `version_template`, as the version
    should not have the leading `v`.

## How it works

When you run GoReleaser with `--nightly`, it will set the `Version` template
variable to the evaluation of `nightly.name_template`. This means that if you
use `{{ .Version }}` on your name templates, you'll get the nightly version.

<!-- md:templates -->

## What is skipped when using `--nightly`?

- Go mod proxying;
- GitHub/GitLab/Gitea releases (unless specified);
- Homebrew taps;
- Scoop manifests;
- Arch User Repositories;
- Krew Plugin Manifests;
- NURs;
- Milestone closing;
- All announcers;

Everything else is executed normally.
Just make sure to use the `Version` template variable instead of `Tag`.
You can also check if it is a nightly build inside a template with:

```
{{ if .IsNightly }}something{{ else }}something else{{ end }}
```

!!! info "Maybe you are looking for something else?"

    - If just want to build the binaries, and no packages at all, check the
      [`goreleaser build` command](../cmd/goreleaser_build.md);
    - If you actually want to create a local "snapshot" build, check out the
      [snapshots documentation](snapshots.md).
