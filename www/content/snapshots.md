---
title: Snapshots
series: customization
hideFromIndex: true
weight: 70
---

Sometimes we want to generate a full build of our project,
but neither want to validate anything nor upload it to anywhere.
GoReleaser supports this with the `--snapshot` flag
and also with the `snapshot` customization section:

```yml
# .goreleaser.yml
snapshot:
  # Allows you to change the name of the generated snapshot
  # Default is `SNAPSHOT-{{.Commit}}`.
  name_template: SNAPSHOT-{{.Commit}}
```

> Learn more about the [name template engine](/templates).
