---
title: "Dist folder"
weight: 30
---

By default, GoReleaser will create its artifacts in the `./dist` folder.
If you must, you can change it by setting it in the `.goreleaser.yaml` file:

```yaml {filename=".goreleaser.yaml"}
# Default: './dist'.
dist: another-folder-that-is-not-dist
```

More often than not, you won't need to change this.

> [!WARNING]
> If you change this value, and use
> `goreleaser continue`,
> you'll need to specify `--dist` when running it.
