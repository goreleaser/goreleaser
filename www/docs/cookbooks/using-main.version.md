# Using the `main.version` ldflag

Defaults-wise GoReleaser sets three _ldflags_:

- `main.version`: Current Git tag (the `v` prefix is stripped) or the name of the snapshot, if you're using the `--snapshot` flag
- `main.commit`: Current git commit SHA
- `main.date`: Date according [RFC3339](https://golang.org/pkg/time/#pkg-constants)

You can use them in your `main.go` file to print more build details:

```go
package main

import "fmt"

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
    builtBy = "unknown"
)

func main() {
  fmt.Printf("my app %s, commit %s, built at %s by %s", version, commit, date, builtBy)
}
```

You can override this by changing the `ldflags` option in the [`build` section](/customization/build/).
