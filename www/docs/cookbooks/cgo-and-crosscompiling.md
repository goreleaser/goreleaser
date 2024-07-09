# Cross-compiling Go with CGO

The best option to cross-compile Go project with CGO dependencies would be in
using Docker image.
[This project](https://github.com/goreleaser/goreleaser-cross) provides the
[Docker images](https://hub.docker.com/repository/docker/goreleaser/goreleaser-cross)
with a bunch of ready-to-use cross-compilers as well as how-to make a `sysroot`.
All that wrapped into [this example](https://github.com/goreleaser/goreleaser-cross-example)

If you have [GoReleaser Pro](../pro.md), you can also use the split and merge feature
to build for each platform natively and merge the builds later.
Check [this article][1] for an example, and the [documentation here][2].

[1]: https://carlosbecker.com/posts/goreleaser-split-merge/
[2]: ../customization/partial.md
