---
title: Snapshots
series: customization
hideFromIndex: true
weight: 70
---

Sometimes we want to generate a full build of our project,
but they aren't full-tested releases, they are nightlies or snapshots.
GoReleaser supports that within the `--snapshot` flag and the `snapshot`
customization section:

```yml
# .goreleaser.yml
snapshot:
  # Allows you to change the name of the generated snapshot
  # Default is `SNAPSHOT-{{.Commit}}`.
  name_template: SNAPSHOT-{{.Commit}}
  # Allows you to still publish your software as a prerelease on github.
  # Default is false
  publish: true
```

> Learn more about the [name template engine](/templates).
