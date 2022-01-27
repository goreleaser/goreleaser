# Cross-compiling Go with CGO

Best option to cross-compile Go project with CGO dependencies would be in using Docker image.
[This project](https://github.com/goreleaser/goreleaser-cross) provides the docker [images](https://hub.docker.com/repository/docker/goreleaser/goreleaser-cross) with bunch of ready to use cross-compilers as well as how-to make sysroot.
All that wrapped into [example](https://github.com/goreleaser/goreleaser-cross-example)
