# Building Go modules

With the default configs, you can already build a Go module without issues.

But, if you want to access module information in runtime (e.g. `debug.BuildInfo` or `go version -m $binary`), you'll
need to setup GoReleaser to "proxy" that module before building it.

To do that, you can simply add this to your config:

```yaml
# goreleaser.yml
gomod:
  proxy: true
```

In practice, what this does is:

- for each of your builds, create a `dist/proxy/$BUILD_ID`;
- creates a `go.mod` that requires your root module at the _current tag_;
- creates a `main.go` that imports your main module;
- copy the projects `go.sum` to that folder.

In which:

- _root module_: is the output of `go list -m`;
- _main module_: is the _root module_ + your build's `main`;
- _current tag_: is the tag that is being built.

So, let's say:

- _root module_: `github.com/goreleaser/nfpm/v2`;
- build's `main`: `./cmd/nfpm/`;
- _current tag_: `v2.5.0`.

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

This will resolve the source code from the defined module proxy.

## Limitations

1. Extra files will still be copied from the current project's root folder and not from the proxy cache;
1. You can't build modules that are not your current module

## More information

You can find more information about it on the [issue][issue] that originated it and its subsequent [pull request][pr].

Make sure to also read the [relevant documentation][docs] for more options.

[issue]: https://github.com/goreleaser/goreleaser/issues/1354
[pr]: https://github.com/goreleaser/goreleaser/pull/2129
[docs]: /customization/gomod/

## Real example

Source code of a working example can be found at [goreleaser/example-mod-proxy](https://github.com/goreleaser/example-mod-proxy).
