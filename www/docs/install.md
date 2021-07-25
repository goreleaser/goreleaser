# Install

There are two GoReleaser distributions: OSS and [Pro](/pro/).

You can install the pre-compiled binary (in several different ways), use Docker or compile from source (when on OSS).

Bellow you can find the steps for each of them.

## Install the pre-compiled binary

### homebrew tap

=== "OSS"
    ```sh
    brew install goreleaser/tap/goreleaser
    ```

=== "Pro"
    ```sh
    brew install goreleaser/tap/goreleaser-pro
    ```

### homebrew

=== "OSS"
    ```sh
    brew install goreleaser
    ```

!!! info
    The [formula in homebrew-core](https://github.com/Homebrew/homebrew-core/blob/master/Formula/goreleaser.rb) might be slightly outdated.
    Use our homebrew tap to always get the latest updates.

### snapcraft

=== "OSS"
    ```sh
    sudo snap install --classic goreleaser
    ```

### scoop

=== "OSS"
    ```sh
    scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
    scoop install goreleaser
    ```

=== "Pro"
    ```sh
    scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
    scoop install goreleaser-pro
    ```

### apt

=== "OSS"
    ```sh
    echo 'deb [trusted=yes] https://apt.fury.io/goreleaser/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
    sudo apt update
    sudo apt install goreleaser
    ```

=== "Pro"
    ```sh
    echo 'deb [trusted=yes] https://apt.fury.io/goreleaser/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
    sudo apt update
    sudo apt install goreleaser-pro
    ```

### yum

=== "OSS"
    ```sh
    echo '[goreleaser]
    name=GoReleaser
    baseurl=https://yum.fury.io/goreleaser/
    enabled=1
    gpgcheck=0' | sudo tee /etc/yum.repos.d/goreleaser.repo
    sudo yum install goreleaser
    ```

=== "Pro"
    ```sh
    echo '[goreleaser]
    name=GoReleaser
    baseurl=https://yum.fury.io/goreleaser/
    enabled=1
    gpgcheck=0' | sudo tee /etc/yum.repos.d/goreleaser.repo
    sudo yum install goreleaser-pro
    ```

### deb, rpm and apk packages


=== "OSS"
    Download the `.deb`, `.rpm` or `.apk` packages from the [OSS releases page][releases] and install them with the appropriate tools.

=== "Pro"
    Download the `.deb`, `.rpm` or `.apk` packages from the [Pro releases page][pro-releases] and install them with the appropriate tools.


### shell script

=== "OSS"
    ```sh
    curl -sfL https://install.goreleaser.com/github.com/goreleaser/goreleaser.sh | sh
    ```


<!-- TODO: write a new shell script and store it within the website -->

### go install

=== "OSS"
    ```sh
    go install github.com/goreleaser/goreleaser@latest
    ```

### manually

=== "OSS"
    Download the pre-compiled binaries from the [OSS releases page][releases] and copy them to the desired location.

=== "Pro"
    Download the pre-compiled binaries from the [Pro releases page][pro-releases] and copy them to the desired location.

## Running with Docker

You can also use it within a Docker container.
To do that, you'll need to execute something more-or-less like the examples bellow.

=== "OSS"
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

=== "Pro"
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
