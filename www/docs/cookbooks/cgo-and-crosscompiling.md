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
When you build with CGO, you inherit a new dependency from C: [libc][3].
This means that your binary will only run on the target [libc][3] (glibc, musl, ect.).
If you want your Linux builds to run on all Linux, you will need to statically link
a libc. You can take a look at [a setup that builds Linux, macOS, and Windows
for amd64 and arm64 with static linking for Linux][4].

[1]: https://carlosbecker.com/posts/goreleaser-split-merge/
[2]: ../customization/partial.md
[3]: https://www.man7.org/linux/man-pages/man7/libc.7.html
[4]: https://github.com/tsukinoko-kun/goreleaser-cgo
