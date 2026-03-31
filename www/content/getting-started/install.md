---
title: "Install"
weight: 20
---

There are two GoReleaser distributions: OSS and [Pro](/pro/), each have a
multitude of installation options.

You can see the instructions for each of them below.

## Homebrew Tap

{{< tabs >}}
{{< tab "OSS" >}}

```bash
brew install --cask goreleaser/tap/goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
brew install --cask goreleaser/tap/goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Homebrew

<br>
{{< badge content="Community Owned" icon="external-link" >}}

{{< tabs >}}
{{< tab "OSS" >}}

```bash
brew install goreleaser
```

> [!WARNING]
> The formula in homebrew-core might be slightly outdated.
> Use our homebrew tap to always get the latest updates.

{{< /tab >}}
{{< tab "Pro" >}}

Not available.
{{< /tab >}}
{{< /tabs >}}

[formula in homebrew-core]: https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/goreleaser.rb

## NPM

{{< tabs >}}
{{< tab "OSS" >}}

```bash
npm i -g @goreleaser/goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
npm i -g @goreleaser/goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Snapcraft

{{< tabs >}}
{{< tab "OSS" >}}

```bash
sudo snap install --classic goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

Not available.
{{< /tab >}}
{{< /tabs >}}

## Scoop

{{< tabs >}}
{{< tab "OSS" >}}

```bash
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Chocolatey

<br>
{{< badge content="Community Owned" icon="external-link" >}}

{{< tabs >}}
{{< tab "OSS" >}}

```bash
choco install goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}
Not available.
{{< /tab >}}
{{< /tabs >}}

## Winget

{{< tabs >}}
{{< tab "OSS" >}}

```bash
winget install goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
winget install goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Apt Repository

{{< tabs >}}
{{< tab "OSS" >}}

```bash
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
sudo apt install goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
sudo apt install goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Yum Repository

{{< tabs >}}
{{< tab "OSS" >}}

```bash
echo '[goreleaser]
name=GoReleaser
baseurl=https://repo.goreleaser.com/yum/
enabled=1
gpgcheck=0
exclude=goreleaser-pro' | sudo tee /etc/yum.repos.d/goreleaser.repo
sudo yum install goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
echo '[goreleaser]
name=GoReleaser
baseurl=https://repo.goreleaser.com/yum/
enabled=1
gpgcheck=0
exclude=goreleaser' | sudo tee /etc/yum.repos.d/goreleaser.repo
sudo yum install goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## AUR

{{< tabs >}}
{{< tab "OSS" >}}

```bash
yay -S goreleaser-bin
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
yay -S goreleaser-pro-bin
```

{{< /tab >}}
{{< /tabs >}}

## Nixpkgs

<br>
{{< badge content="Community Owned" icon="external-link" >}}

{{< tabs >}}
{{< tab "OSS" >}}

```bash
nix-shell -p goreleaser
```

> [!WARNING]
> The package in nixpkgs might be slightly outdated, as it is not
> updated automatically.
> Use our NUR to always get the latest updates.

{{< /tab >}}
{{< tab "Pro" >}}

Not available.
{{< /tab >}}
{{< /tabs >}}

[package in nixpkgs]: https://github.com/NixOS/nixpkgs/blob/master/pkgs/tools/misc/goreleaser/default.nix

## NUR

First, you'll need to add our [NUR][nur] to your nix configuration.
You can follow the guides
[here](https://github.com/nix-community/NUR#installation).

Once you do that, you can install the packages.

[nur]: https://github.com/goreleaser/nur

{{< tabs >}}
{{< tab "OSS" >}}

```nix
{ pkgs, lib, ... }: {
  home.packages = with pkgs; [
    nur.repos.goreleaser.goreleaser
  ];
}
```

{{< /tab >}}
{{< tab "Pro" >}}

```nix
{ pkgs, lib, ... }: {
  home.packages = with pkgs; [
    nur.repos.goreleaser.goreleaser-pro
  ];
}
```

{{< /tab >}}
{{< /tabs >}}

## Docker

{{< tabs >}}
{{< tab "OSS" >}}

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

{{< /tab >}}
{{< tab "Pro" >}}

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

{{< /tab >}}
{{< /tabs >}}

> [!WARNING]
> The provided docker image does not support the Snapcraft feature.

The `DOCKER_REGISTRY` environment variable can be left empty when you are
releasing to the public docker registry.

If you need more things, you are encouraged to keep your own image. You can
always use GoReleaser's [own Dockerfile][dockerfile] as an example though
and iterate from that.

> [!NOTE]
> There are also `:nightly` tags available with the latest nightly builds.

## Linux packages

{{< tabs >}}
{{< tab "OSS" >}}

Download the `.deb`, `.rpm`, or `.apk` packages from the releases page and
install them with the appropriate tools.

{{% button href="https://github.com/goreleaser/goreleaser/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

{{< /tab >}}
{{< tab "Pro" >}}

Download the `.deb`, `.rpm`, or `.apk` packages from the releases page and
install them with the appropriate tools.

{{% button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

{{< /tab >}}
{{< /tabs >}}

To install, after downloading the files, run:

```bash
dpkg -i goreleaser*.deb
rpm -ivh goreleaser*.rpm
apk add --allow-untrusted goreleaser*.apk
```

## `go install`

{{< tabs >}}
{{< tab "OSS" >}}

```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

Requires Go 1.26.
{{< /tab >}}
{{< tab "Pro" >}}

Not available.
{{< /tab >}}
{{< /tabs >}}

## Bash Script

This script does not install anything, it just downloads, verifies and runs
GoReleaser.
Its purpose is to be used within scripts and CIs.

{{< tabs >}}
{{< tab "OSS" >}}

```bash
curl -sfL https://goreleaser.com/static/run | bash VERSION=__VERSION__ -s -- check
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
curl -sfL https://goreleaser.com/static/run | DISTRIBUTION=pro VERSION=__VERSION__ bash -s -- check
```

{{< /tab >}}
{{< /tabs >}}

> [!NOTE]
> The `VERSION` environment variable can be omitted to get the latest stable
> version, or you can set it to `nightly` to get the last nightly build.

## Manually

{{< tabs >}}
{{< tab "OSS" >}}

Download the pre-compiled binaries from the releases page and copy them to the
desired location:

{{% button href="https://github.com/goreleaser/goreleaser/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

{{< /tab >}}
{{< tab "Pro" >}}

Download the pre-compiled binaries from the releases page and copy them to the
desired location:

{{% button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

{{< /tab >}}
{{< /tabs >}}

## Verifying the artifacts

### Binaries

#### Signatures

All artifacts are checksummed, and the checksum file is signed with [cosign][].

{{< tabs >}}
{{< tab "OSS" >}}

1. Download the files you want, and the `checksums.txt`, `checksum.txt.sigstore.json` files from the
   [releases](https://github.com/goreleaser/goreleaser/releases) page:

```bash
wget 'https://github.com/goreleaser/goreleaser/releases/download/__VERSION__/checksums.txt'
wget 'https://github.com/goreleaser/goreleaser/releases/download/__VERSION__/checksums.txt.sigstore.json'
```

1. Verify the signature:

```bash
cosign verify-blob \
  --certificate-identity 'https://github.com/goreleaser/goreleaser/.github/workflows/release.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --bundle checksums.txt.sigstore.json \
  ./checksums.txt
```

1. If the signature is valid, you can then verify the SHA256 sums match with the downloaded binary:

```bash
sha256sum --ignore-missing -c checksums.txt
```

{{< /tab >}}
{{< tab "Pro" >}}

1. Download the files you want, and the `checksums.txt`, `checksum.txt.sigstore.json` files from the
   [releases](https://github.com/goreleaser/goreleaser-pro/releases) page:

```bash
wget 'https://github.com/goreleaser/goreleaser-pro/releases/download/__VERSION__/checksums.txt'
wget 'https://github.com/goreleaser/goreleaser-pro/releases/download/__VERSION__/checksums.txt.sigstore.json'
```

1. Verify the signature:

```bash
cosign verify-blob \
  --certificate-identity 'https://github.com/goreleaser/goreleaser-pro-internal/.github/workflows/release-pro.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --bundle checksums.txt.sigstore.json \
  ./checksums.txt
```

1. If the signature is valid, you can then verify the SHA256 sums match with the downloaded binary:

```bash
sha256sum --ignore-missing -c checksums.txt
```

{{< /tab >}}
{{< /tabs >}}

#### Attestations

You can also verify the attestations:

{{< tabs >}}
{{< tab "OSS" >}}

```bash
gh attestation verify --owner goreleaser *.tar.gz
# PS: can be any file from the release
```

{{< /tab >}}
{{< tab "Pro" >}}

GitHub does not yet allow cross-repository attestations (e.g. building a
private repo and publishing the attestations in a public one), so this isn't
available yet, unfortunately.
{{< /tab >}}
{{< /tabs >}}

### Docker images

Our Docker images are signed with [cosign][].

Verify the signatures:

{{< tabs >}}
{{< tab "OSS" >}}

```bash
cosign verify \
  --certificate-identity 'https://github.com/goreleaser/goreleaser/.github/workflows/release.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  goreleaser/goreleaser
```

{{< /tab >}}
{{< tab "Pro" >}}

```bash
cosign verify \
  --certificate-identity 'https://github.com/goreleaser/goreleaser-pro-internal/.github/workflows/release-pro.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  goreleaser/goreleaser-pro
```

{{< /tab >}}
{{< /tabs >}}

## Nightly builds

Nightly build are pre-releases of the current code into the main branch.
Use it for testing out new features only.

{{< tabs >}}
{{< tab "OSS" >}}

Download the pre-compiled binaries from the
[nightly release](https://github.com/goreleaser/goreleaser/releases/nightly)
and copy them to the desired location.
{{< /tab >}}
{{< tab "Pro" >}}

Download the pre-compiled binaries from the
[nightly release](https://github.com/goreleaser/goreleaser-pro/releases/nightly)
and copy them to the desired location.

{{< /tab >}}
{{< /tabs >}}

[Docker](#docker) images are also available, look for tags with a `-nightly`
suffix for the last nightly of a specific release, or the `:nightly` tag,
which is always the latest nightly build available.

You may also use the [Bash Script method](#bash-script) by setting the `VERSION`
environment variable to `nightly`.

## Packaging status

[![Packaging status](https://repology.org/badge/vertical-allrepos/goreleaser.svg)](https://repology.org/project/goreleaser/versions)

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/main/Dockerfile
[cosign]: https://github.com/sigstore/cosign

## Community

Install options with the "Community Owned" badge are maintained by the community
and might not always be up to date.
