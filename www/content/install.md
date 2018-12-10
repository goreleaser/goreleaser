---
title: Install
weight: 2
menu: true
---

You can install the pre-compiled binary, use Docker or compile from source.

## Install the pre-compiled binary

**homebrew tap**:

```sh
$ brew install goreleaser/tap/goreleaser
```

**homebrew** (may not be the latest version):

```sh
$ brew install goreleaser
```

**snapcraft**:

```sh
$ snap install goreleaser
```

**scoop**:

```sh
$ scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
$ scoop install goreleaser
```

**deb/rpm**:

Download the `.deb` or `.rpm` from the [releases page][releases] and
install with `dpkg -i` and `rpm -i` respectively.

**manually**:

Download the pre-compiled binaries from the [releases page][releases] and
copy to the desired location.

## Running with Docker

You can use Docker to do simple releases. Currently, the provided docker
image does not provide support for snapcraft.

```sh
$ docker run --rm --privileged \
  -v $PWD:/go/src/github.com/user/repo \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /go/src/github.com/user/repo \
  -e GITHUB_TOKEN \
  -e DOCKER_USERNAME \
  -e DOCKER_PASSWORD \
  -e DOCKER_REGISTRY \
  goreleaser/goreleaser release
```

Note that the image will almost always have the last stable Go version.

The `DOCKER_REGISTRY` environment variables can be left empty when you are
releasing to the public docker registry.

If you need more things, you are encouraged to have your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/master/Dockerfile
[releases]: https://github.com/goreleaser/goreleaser/releases

## Compiling from source

> **Note**: this method requires Go 1.11+.

```sh
$ git clone git@github.com:goreleaser/goreleaser.git
$ cd goreleaser
$ make setup build
```

After that, the `goreleaser` binary will be in the root folder:

```sh
$ ./goreleaser --help
```

For more information, check the [contributing guide][contrib].

[contrib]: /contributing
