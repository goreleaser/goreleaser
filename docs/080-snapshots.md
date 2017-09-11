---
title: Snapshots
---

Sometimes we want to generate a full build of our project for some reason,
but don't want to validate anything nor upload it to anywhere.
GoRelease supports this with a `--snapshot` flag and with a `snapshot`
customization section as well.

```yml
# .goreleaser.yml
snapshot:
  # Allows you to change the name of the generated snapshot
  # releases. The following variables are available:
  # - Commit
  # - Tag
  # - Timestamp
  # Default: SNAPSHOT-{{.Commit}}
  name_template: SNAPSHOT-{{.Commit}}
```
