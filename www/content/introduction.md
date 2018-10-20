---
title: Introduction
weight: 1
menu: true
---

[GoReleaser](https://github.com/goreleaser/goreleaser) is a release automation
tool for Go projects, the goal is to simplify the build, release and
publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools; you only need to
[download and execute it](#ci_integration) in your build script.
You can [customize](#Customization) your release process by
creating a `.goreleaser.yml` file.

The idea started with a
[simple shell script](https://github.com/goreleaser/old-go-releaser),
but it quickly became more complex and I also wanted to publish binaries via
Homebrew taps, which would have made the script even more hacky, so I let go of
that and rewrote the whole thing in Go.

## Installing Goreleaser

There are three ways to get going install GoReleaser:

### Using homebrew

```sh
brew install goreleaser/tap/goreleaser
```

### Using snapcraft

```sh
snap install goreleaser
```

### Using Scoop

```sh
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser
```

> Check the [tap source](https://github.com/goreleaser/homebrew-tap) for
> more details.

### Using Docker

You can use Docker to do simple releases. Currently, the provided docker
image does not provide support for snapcraft.

```console
$ docker run --rm --privileged \
  -v $PWD:/go/src/github.com/user/repo \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /go/src/github.com/user/repo \
  -e GITHUB_TOKEN \
  -e DOCKER_USERNAME \
  -e DOCKER_PASSWORD \
  goreleaser/goreleaser release
```

Note that the image will almost always have the last stable Go version.

If you need more things, you are encouraged to have your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/master/Dockerfile

## Manually

Download your preferred flavor from the [releases page](https://github.com/goreleaser/goreleaser/releases/latest) and install
manually.

### Using go get

Note: this method requires Go 1.10+.

```console
$ go get -d github.com/goreleaser/goreleaser
$ cd $GOPATH/src/github.com/goreleaser/goreleaser
$ dep ensure -vendor-only
$ make setup build
```

It is recommended to also run `dep ensure` to make sure that the dependencies
are in the correct versions.
