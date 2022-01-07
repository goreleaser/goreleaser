# Building Go modules

With the default configs, you can already build a Go module without issues.

But, if you want to access module information in runtime (e.g. `debug.BuildInfo` or `go version -m $binary`), you'll
need to setup GoReleaser to "proxy" that module before building it.

To do that, you can simply add this to your config:

```yaml
# goreleaser.yaml
gomod:
  proxy: true
```

In practice, what this does is:

- for each of your builds, create a `dist/proxy/{{ build.id }}`;
- creates a `go.mod` that requires your __main module__ at the __current tag__;
- creates a `main.go` that imports your __main package__;
- copy the project's `go.sum` to that folder.

In which:

- __build.id__: the `id` property in your `build` definition;
- __main module__: is the output of `go list -m`;
- __main package__: is the __main module__ + your build's `main`;
- __current tag__: is the tag that is being built.

So, let's say:

- __main module__: `github.com/goreleaser/nfpm/v2`;
- build's `main`: `./cmd/nfpm/`;
- __current tag__: `v2.5.0`.

GoReleaser will create a `main.go` like:

```go
// +build: main
package main

import _ "github.com/goreleaser/nfpm/v2/cmd/nfpm"
```

a `go.mod` like:

```
module nfpm

require github.com/goreleaser/nfpm/v2 v2.5.0
```

Then, it'll run:

```sh
go mod tidy
```

And, to build, it will use something like:

```shell
go build -o nfpm github.com/goreleaser/nfpm/v2/cmd/nfpm
```

This will resolve the source code from the defined module proxy using `proxy.golang.org`.
Your project's `go.sum` will be used to verify any modules that are downloaded, with `sum.golang.org` "filling in" any gaps.

## Getting the version from the module

You can also get the module version at runtime using [`debug#ReadBuildInfo`](https://pkg.go.dev/runtime/debug#ReadBuildInfo).
It is useful to display the version of your program to the user, for example.

```golang
package main

import (
	"fmt"
	"runtime/debug"
)

func main() {
  if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
    fmt.Println(info)
  }
}
```

You can also use `go version -m my_program` to display the go module information.

## Limitations

1. Extra files will still be copied from the current project's root folder and not from the proxy cache;
1. You can't build packages that are not contained in the main module.

## More information

You can find more information about it on the [issue][issue] that originated it and its subsequent [pull request][pr].

Make sure to also read the [relevant documentation][docs] for more options.

[issue]: https://github.com/goreleaser/goreleaser/issues/1354
[pr]: https://github.com/goreleaser/goreleaser/pull/2129
[docs]: /customization/gomod/

## Real example

Source code of a working example can be found at [goreleaser/example-mod-proxy](https://github.com/goreleaser/example-mod-proxy).
