# GoReleaser XX

[`goreleaser-xx`](https://github.com/crazy-max/goreleaser-xx#readme) is a small
CLI wrapper for GoReleaser and available as a [lightweight and multi-platform scratch Docker image](https://hub.docker.com/r/crazymax/goreleaser-xx/tags?page=1&ordering=last_updated)
to ease the integration and cross compilation in a Dockerfile for your Go
projects using [Buildx](https://github.com/docker/buildx) Docker component that
enables many powerful build features with [Moby BuildKit](https://github.com/moby/buildkit)
builder engine.

- Handle `--platform` in your Dockerfile for multi-platform support
- Build into any architecture
- Handle C and C++ compilers for [CGO dependencies](https://github.com/crazy-max/goreleaser-xx#cgo)
- Translation of [platform ARGs in the global scope](https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope) into Go compiler's target
- Auto generation of `.goreleaser.yml` config based on target architecture

Many examples are provided in the [`demo` folder](https://github.com/crazy-max/goreleaser-xx/tree/master/demo)
of the repository.
