# CGO

Compiling with CGO is tricky, especially when cross-compiling. It requires extra
setup and won't work "out of the box".

Here are a few ways you can do it with GoReleaser:

## GoReleaser Pro

If you have [GoReleaser Pro](../pro.md), you can use the split and merge feature
to build for each platform natively and merge the builds later.

This is the recommended approach as it's the simplest and most reliable.

Check [this article][1] for an example, and the [documentation here][2].

## Using Docker

Another option is to use a Docker image with the required cross-compilers.
[This project](https://github.com/goreleaser/goreleaser-cross) provides the
[Docker images](https://hub.docker.com/repository/docker/goreleaser/goreleaser-cross)
with a bunch of ready-to-use cross-compilers as well as how-to make a `sysroot`.
All that wrapped into [this example](https://github.com/goreleaser/goreleaser-cross-example).

## Using Zig

In some cases, you can use Zig to act as a C/C++ compiler, which makes
cross-compilation with CGO easier.

You can find an example of this approach in
[this repository](https://github.com/goreleaser/example-zig-cgo).

This might not work for all cases, but it's a good alternative to explore.

[1]: https://carlosbecker.com/posts/goreleaser-split-merge/
[2]: ../customization/partial.md

## Community tools

Tools like `xgo` are not natively supported, and we make no promises about
whether or how well they work within GoReleaser or not.
