# Nightly

!!! success "GoReleaser Pro"
    The nightly build feature is a [GoReleaser Pro feature](/pro/).

Whether if you need beta builds or a rolling-release system, the nightly builds feature gets you covered.

To enable it, you must use the `--nightly` flag in the [`goreleaser release` command](/cmd/goreleaser_release/).

You also have some customization options available:

```yaml
# .goreleaser.yml
nightly:
  # Allows you to change the version of the generated nightly release.
  #
  # Note that some pipes require this to be semantic version compliant (nfpm, for example).
  #
  # Default is `{{ incpatch .Version }}-{{ .ShortCommit }}-dev`.
  name_template: '{{ incpatch .Version }}-devel'
```

## How it works

When you run GoReleaser with `--nightly`, it will set the `Version` template variable to the evaluation of `nightly.name_template`.
This means that if you use `{{ .Version }}` on your name templates, you'll get the nightly version.

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## What is skipped when using `--nightly`?

- Go mod proxying;
- GitHub/GitLab/Gitea releases;
- Homebrew taps;
- Scoop manifests;
- Milestone closing;
- All announcers;

Everything else is executed normally. Just make sure to use the `Version` template variable instead of `Tag`.
You can also check if its a nightly build inside a template with:

```
{{ if .IsNightly }}something{{ else }}something else{{ end }}
```

!!! info "Maybe you are looking for something else?"
    - If just want to build the binaries, and no packages at all, check the [`goreleaser build` command](/cmd/goreleaser_build/);
    - If you actually want to create a local "snapshot" build, check out the [snapshots documentation](/customization/snapshots/).
