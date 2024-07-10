# Install

There are two GoReleaser distributions: OSS and [Pro](pro.md), each have a
multitude of installation options.

You can see the instructions for each of them below.

## Homebrew Tap

=== "OSS"

    ```bash
    brew install goreleaser/tap/goreleaser
    ```

=== "Pro"

    ```bash
    brew install goreleaser/tap/goreleaser-pro
    ```

## Homebrew

=== "OSS"

    ```bash
    brew install goreleaser
    ```

    !!! warning

        The [formula in homebrew-core] might be slightly outdated.
        Use our homebrew tap to always get the latest updates.

=== "Pro"

    Not available.

[formula in homebrew-core]: https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/goreleaser.rb

## Snapcraft

=== "OSS"

    ```bash
    sudo snap install --classic goreleaser
    ```

=== "Pro"

    Not available.

## Scoop

=== "OSS"

    ```bash
    scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
    scoop install goreleaser
    ```

=== "Pro"

    ```bash
    scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
    scoop install goreleaser-pro
    ```

## Apt Repository

=== "OSS"

    ```bash
    echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
    sudo apt update
    sudo apt install goreleaser
    ```

=== "Pro"

    ```bash
    echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
    sudo apt update
    sudo apt install goreleaser-pro
    ```

## Yum Repository

=== "OSS"

    ```bash
    echo '[goreleaser]
    name=GoReleaser
    baseurl=https://repo.goreleaser.com/yum/
    enabled=1
    gpgcheck=0' | sudo tee /etc/yum.repos.d/goreleaser.repo
    sudo yum install goreleaser
    ```

=== "Pro"

    ```bash
    echo '[goreleaser]
    name=GoReleaser
    baseurl=https://repo.goreleaser.com/yum/
    enabled=1
    gpgcheck=0' | sudo tee /etc/yum.repos.d/goreleaser.repo
    sudo yum install goreleaser-pro
    ```

## AUR

=== "OSS"

    ```bash
    yay -S goreleaser-bin
    ```

=== "Pro"

    ```bash
    yay -S goreleaser-pro-bin
    ```

## Nixpkgs

=== "OSS"

    ```bash
    nix-shell -p goreleaser
    ```

    !!! warning

        The [package in nixpkgs] might be slightly outdated, as it is not
        updated automatically.
        Use our NUR to always get the latest updates.

=== "Pro"

    Not available.

[package in nixpkgs]: https://github.com/NixOS/nixpkgs/blob/master/pkgs/tools/misc/goreleaser/default.nix

## NUR

First, you'll need to add our [NUR][nur] to your nix configuration.
You can follow the guides
[here](https://github.com/nix-community/NUR#installation).

Once you do that, you can install the packages.

[nur]: https://github.com/goreleaser/nur

=== "OSS"

    ```nix
    { pkgs, lib, ... }: {
      home.packages = with pkgs; [
        nur.repos.goreleaser.goreleaser
      ];
    }
    ```

=== "Pro"

    ```nix
    { pkgs, lib, ... }: {
      home.packages = with pkgs; [
        nur.repos.goreleaser.goreleaser-pro
      ];
    }
    ```

## Docker

=== "OSS"

    Registries:

    - [`goreleaser/goreleaser`](https://hub.docker.com/r/goreleaser/goreleaser)
    - [`ghcr.io/goreleaser/goreleaser`](https://github.com/goreleaser/goreleaser/pkgs/container/goreleaser)

    **Example usage:**

    ```bash
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
    - [`ghcr.io/goreleaser/goreleaser-pro`](https://github.com/goreleaser/goreleaser/pkgs/container/goreleaser-pro)

    **Example usage:**

    ```bash
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

!!! warning

    The provided docker image does not support the Snapcraft feature.

The `DOCKER_REGISTRY` environment variable can be left empty when you are
releasing to the public docker registry.

If you need more things, you are encouraged to keep your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though
and iterate from that.

!!! tip

    There are also `:nightly` tags available with the latest nightly builds.

## Linux packages

=== "OSS"

    Download the `.deb`, `.rpm`, or `.apk` packages from the [releases page][releases] and install them with the appropriate tools.

=== "Pro"

    Download the `.deb`, `.rpm`, or `.apk` packages from the [releases page][pro-releases] and install them with the appropriate tools.

To install, after downloading the files, run:

```bash
dpkg -i goreleaser*.deb
rpm -ivh goreleaser*.rpm
apk add --allow-untrusted goreleaser*.apk
```

## `go install`

=== "OSS"

    ```bash
    go install github.com/goreleaser/goreleaser/v2@latest
    ```

    Requires Go 1.22.

=== "Pro"

    Not available.

## Bash Script

This script does not install anything, it just downloads, verifies and runs
GoReleaser.
Its purpose is to be used within scripts and CIs.

=== "OSS"

    ```bash
    curl -sfL https://goreleaser.com/static/run | bash VERSION=__VERSION__ -s -- check
    ```

=== "Pro"

    ```bash
    curl -sfL https://goreleaser.com/static/run | DISTRIBUTION=pro VERSION=__VERSION__ bash -s -- check
    ```

!!! tip

    The `VERSION` environment variable can be ommited to get the latest stable
    version, or you can set it to `nightly` to get the last nightly build.

## Manually

=== "OSS"

    Download the pre-compiled binaries from the [releases page][releases] and copy them to the desired location.

=== "Pro"

    Download the pre-compiled binaries from the [releases page][pro-releases] and copy them to the desired location.

## Verifying the artifacts

### Binaries

All artifacts are checksummed, and the checksum file is signed with [cosign][].

=== "OSS"

    1. Download the files you want, and the `checksums.txt`, `checksum.txt.pem` and `checksums.txt.sig` files from the [releases][releases] page:
      ```bash
      wget 'https://github.com/goreleaser/goreleaser/releases/download/__VERSION__/checksums.txt'
      ```
    1. Verify the signature:
      ```bash
      cosign verify-blob \
        --certificate-identity 'https://github.com/goreleaser/goreleaser/.github/workflows/release.yml@refs/tags/__VERSION__' \
        --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
        --cert 'https://github.com/goreleaser/goreleaser/releases/download/__VERSION__/checksums.txt.pem' \
        --signature 'https://github.com/goreleaser/goreleaser/releases/download/__VERSION__/checksums.txt.sig' \
        ./checksums.txt
      ```
    1. If the signature is valid, you can then verify the SHA256 sums match with the downloaded binary:
      ```bash
      sha256sum --ignore-missing -c checksums.txt
      ```

=== "Pro"

    1. Download the files you want, and the `checksums.txt`, `checksum.txt.pem` and `checksums.txt.sig` files from the [releases][pro-releases] page:
      ```bash
      wget 'https://github.com/goreleaser/goreleaser-pro/releases/download/__VERSION__-pro/checksums.txt'
      ```
    1. Verify the signature:
      ```bash
      cosign verify-blob \
        --certificate-identity 'https://github.com/goreleaser/goreleaser-pro-internal/.github/workflows/release-pro.yml@refs/tags/__VERSION__-pro' \
        --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
        --cert 'https://github.com/goreleaser/goreleaser-pro/releases/download/__VERSION__-pro/checksums.txt.pem' \
        --signature 'https://github.com/goreleaser/goreleaser-pro/releases/download/__VERSION__-pro/checksums.txt.sig' \
        ./checksums.txt
      ```
    1. If the signature is valid, you can then verify the SHA256 sums match with the downloaded binary:
      ```bash
      sha256sum --ignore-missing -c checksums.txt
      ```

### Docker images

Our Docker images are signed with [cosign][].

Verify the signatures:

=== "OSS"

    ```bash
    cosign verify \
      --certificate-identity 'https://github.com/goreleaser/goreleaser/.github/workflows/release.yml@refs/tags/__VERSION__' \
      --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
      goreleaser/goreleaser
    ```

=== "Pro"

    ```bash
    cosign verify \
      --certificate-identity 'https://github.com/goreleaser/goreleaser-pro-internal/.github/workflows/release-pro.yml@refs/tags/__VERSION__-pro' \
      --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
      goreleaser/goreleaser-pro
    ```

!!! info

    The `.pem` and `.sig` files are the image `name:tag`, replacing `/` and `:` with `-`.

## Nightly builds

Nightly build are pre-releases of the current code into the main branch.
Use it for testing out new features only.

=== "OSS"

    Download the pre-compiled binaries from the [nightly release][nightly-releases] and copy them to the desired location.

=== "Pro"

    Download the pre-compiled binaries from the [nightly release][nightly-pro-releases] and copy them to the desired location.

[Docker](#docker) images are also available, look for tags with a `-nightly`
suffix for the last nightly of a specific release, or the `:nightly` tag,
which is always the latest nightly build available.

You may also use the [Bash Script method](#bash-script) by setting the `VERSION`
environment variable to `nightly`.

## Packaging status

[![Packaging status](https://repology.org/badge/vertical-allrepos/goreleaser.svg)](https://repology.org/project/goreleaser/versions)

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/main/Dockerfile
[releases]: https://github.com/goreleaser/goreleaser/releases
[pro-releases]: https://github.com/goreleaser/goreleaser-pro/releases
[nightly-pro-releases]: https://github.com/goreleaser/goreleaser-pro/releases/nightly
[nightly-releases]: https://github.com/goreleaser/goreleaser/releases/nightly
[cosign]: https://github.com/sigstore/cosign
