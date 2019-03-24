---
title: Install
weight: 2
menu: true
---

You can install the pre-compiled binary (in several different ways),
use Docker or compile from source.

Here are the steps for each of them:

## Install the pre-compiled binary

**homebrew tap** (only on macOS for now):

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

You can also use it within a Docker container. To do that, you'll need to
execute something more-or-less like the following:

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

> **ATTENTION**: Currently, the provided docker image does not support
> the generation of snapcraft packages.

Note that the image will almost always have the last stable Go version.

The `DOCKER_REGISTRY` environment variable can be left empty when you are
releasing to the public docker registry.

If you need more things, you are encouraged to keep your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though
and iterate from that.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/master/Dockerfile
[releases]: https://github.com/goreleaser/goreleaser/releases

## Compiling from source

Please follow the relevant steps on our [contributing guide](/contributing).
