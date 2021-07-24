---
title: Snapshots
---

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
  # Default is `{{ .Tag }}-SNAPSHOT-{{.ShortCommit}}`.
  name_template: '{{ incpatch .Tag }}-devel'
```

## How it works

When you run GoReleaser with `--snapshot`, it will set the `Version` template
variable to the evaluation of `snapshot.name_template`.

This means that if you use `{{ .Version }}` on your name templates, you'll
get the snapshot version.

!!! tip
    Learn more about the [name template engine](/customization/templates/).

Note that the idea behind GoReleaser's snapshots if mostly for local builds
or to validate your build on the CI pipeline. Artifacts shouldn't be uploaded
anywhere, and will only be generated to the `dist` folder.
