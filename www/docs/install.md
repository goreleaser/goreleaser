# Install

There are two GoReleaser distributions: OSS and [Pro](/pro/).

You can install the pre-compiled binary (in several different ways), use Docker or compile from source (when on OSS).

Here are the steps for each of them:

## Install the pre-compiled binary

### homebrew tap

#### oss

```sh
brew install goreleaser/tap/goreleaser
```

#### pro

```sh
brew install goreleaser/tap/goreleaser-pro
```

### homebrew

OSS-only, may not be the latest version.

```sh
brew install goreleaser
```

### snapcraft

OSS only.

```sh
sudo snap install --classic goreleaser
```

### scoop

#### oss

```sh
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser
```

#### pro

```sh
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser-pro
```

### apt

#### setup repository

```sh
echo 'deb [trusted=yes] https://apt.fury.io/goreleaser/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
```

#### oss

```sh
sudo apt install goreleaser
```

#### pro

```sh
sudo apt install goreleaser-pro
```

### yum

#### setup repository

```sh
echo '[goreleaser]
name=GoReleaser
baseurl=https://yum.fury.io/goreleaser/
enabled=1
gpgcheck=0' | sudo tee /etc/yum.repos.d/goreleaser.repo
```

#### oss

```sh
sudo yum install goreleaser
```

#### pro

```sh
sudo yum install goreleaser-pro
```

### deb, rpm and apk packages

Download the `.deb`, `.rpm` or `.apk` from the [OSS][releases] or [Pro][pro-releases] releases pages and install them with the appropriate tools.

### shell script

OSS only.

```sh
curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh
```

<!-- TODO: write a new shell script and store it within the website -->

### go install

OSS only.

```sh
go install github.com/goreleaser/goreleaser
```

### manually

Download the pre-compiled binaries from the [OSS][releases] or [Pro][pro-releases] releases pages and copy to the desired location.

## Running with Docker

You can also use it within a Docker container.
To do that, you'll need to execute something more-or-less like the examples bellow.

### oss

Registries:

- [`goreleaser/goreleaser`](https://hub.docker.com/r/goreleaser/goreleaser)
- [`ghcr.io/goreleaser/goreleaser`](https://github.com/orgs/goreleaser/packages/container/package/goreleaser)

Example usage:

```sh
docker run --rm --privileged \
  -v $PWD:/go/src/github.com/user/repo \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /go/src/github.com/user/repo \
  -e GITHUB_TOKEN \
  -e DOCKER_USERNAME \
  -e DOCKER_PASSWORD \
  -e DOCKER_REGISTRY \
  goreleaser/goreleaser release
```

### pro

Registries:

- [`goreleaser/goreleaser-pro`](https://hub.docker.com/r/goreleaser/goreleaser-pro)
- [`ghcr.io/goreleaser/goreleaser-pro`](https://github.com/orgs/goreleaser/packages/container/package/goreleaser-pro)

Example usage:

```sh
docker run --rm --privileged \
  -v $PWD:/go/src/github.com/user/repo \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -w /go/src/github.com/user/repo \
  -e GITHUB_TOKEN \
  -e DOCKER_USERNAME \
  -e DOCKER_PASSWORD \
  -e DOCKER_REGISTRY \
  -e GORELEASER_KEY \
  goreleaser/goreleaser-pro release
```

!!! info
    Currently, the provided docker image does not support
    the generation of snapcraft packages.

Note that the image will almost always have the last stable Go version.

The `DOCKER_REGISTRY` environment variable can be left empty when you are
releasing to the public docker registry.

If you need more things, you are encouraged to keep your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though
and iterate from that.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/master/Dockerfile
[releases]: https://github.com/goreleaser/goreleaser/releases
[pro-releases]: https://github.com/goreleaser/goreleaser-pro/releases

## Compiling from source

Here you have two options:

If you want to contribute to the project, please follow the
steps on our [contributing guide](/contributing/).

If you just want to build from source for whatever reason, follow these steps:

**clone:**

```sh
git clone https://github.com/goreleaser/goreleaser
cd goreleaser
```

**get the dependencies:**

```sh
go mod tidy
```

**build:**

```sh
go build -o goreleaser .
```

**verify it works:**

```sh
./goreleaser --version
```
