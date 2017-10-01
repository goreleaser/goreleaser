---
title: main.version
---

GoReleaser always sets a `main.version` *ldflag*.
You can use it in your `main.go` file:

```go
package main

var version = "master"

func main() {
  println(version)
}
```

`version` will be set to the current Git tag (the `v` prefix is stripped) or the name of
the snapshot, if you're using the `--snapshot` flag.

You can override this by changing the `ldflags` option in the `build` section.
