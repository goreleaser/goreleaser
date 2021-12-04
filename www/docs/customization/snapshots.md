# Snapshots

Sometimes we want to generate a full build of our project,
but neither want to validate anything nor upload it to anywhere.

GoReleaser supports this with the `--snapshot` flag
and also with the `snapshot` customization section:

```yaml
# .goreleaser.yml
snapshot:
  # Allows you to change the name of the generated snapshot
  #
  # Note that some pipes require this to be semantic version compliant (nfpm,
  # for example).
  #
  # Default is `{{ .Version }}-SNAPSHOT-{{.ShortCommit}}`.
  name_template: '{{ incpatch .Version }}-devel'
```

## How it works

When you run GoReleaser with `--snapshot`, it will set the `Version` template variable to the evaluation of `snapshot.name_template`.
This means that if you use `{{ .Version }}` on your name templates, you'll get the snapshot version.

You can also check if its a snapshot build inside a template with:

```
{{ if .IsSnapshot }}something{{ else }}something else{{ end }}
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

Note that the idea behind GoReleaser's snapshots is for local builds or to validate your build on the CI pipeline.
Artifacts wont't be uploaded and will only be generated into the `dist` folder.

!!! info "Maybe you are looking for something else?"
    - If just want to build the binaries, and no packages at all, check the [`goreleaser build` command](/cmd/goreleaser_build/);
    - If you actually want to create nightly builds, check out the [nightly documentation](/customization/nightly/).
