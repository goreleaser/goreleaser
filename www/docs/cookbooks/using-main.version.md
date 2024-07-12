# Using the `main.version` ldflag

By default, GoReleaser will set the following 3 _ldflags_:

- `main.version`: Current Git tag (the `v` prefix is stripped) or the name of
  the snapshot, if you're using the `--snapshot` flag
- `main.commit`: Current git commit SHA
- `main.date`: Date in the
  [RFC3339](https://pkg.go.dev/time#pkg-constants) format

You can use them in your `main.go` file to print more build details:

```go
package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
  fmt.Printf("my app %s, commit %s, built at %s", version, commit, date)
}
```

You can override this by changing the `ldflags` option in the
[`build` section](../customization/builds.md).
