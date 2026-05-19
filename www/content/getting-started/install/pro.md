---
title: "Install GoReleaser Pro"
linkTitle: Pro
weight: 20
description: "Installing the GoReleaser Pro distribution."
---

{{< cards cols="2" >}}
{{< card link="https://github.com/goreleaser/goreleaser-pro/releases/tag/__VERSION__" title="Latest stable" subtitle="`__VERSION__`" >}}
{{< card link="https://github.com/goreleaser/goreleaser-pro/releases/nightly" title="Latest nightly" subtitle="<span data-nightly-tag data-repo='goreleaser/goreleaser-pro'>loading…</span>" >}}
{{< /cards >}}

See all releases on [GitHub](https://github.com/goreleaser/goreleaser-pro/releases).

{{< g_install_versions >}}

## Homebrew Tap

```bash
brew install --cask goreleaser/tap/goreleaser-pro
```

## NPM

```bash
npm i -g @goreleaser/goreleaser-pro
```

## Scoop

```bash
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser-pro
```

## Winget

```bash
winget install goreleaser-pro
```

## Apt Repository

```bash
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
sudo apt install goreleaser-pro
```

## Yum Repository

```bash
echo '[goreleaser]
name=GoReleaser
baseurl=https://repo.goreleaser.com/yum/
enabled=1
gpgcheck=0
exclude=goreleaser' | sudo tee /etc/yum.repos.d/goreleaser.repo
sudo yum install goreleaser-pro
```

## AUR

```bash
yay -S goreleaser-pro-bin
```

## NUR

First, you'll need to add our [NUR][nur] to your nix configuration.
You can follow the guides
[here](https://github.com/nix-community/NUR#installation).

Once you do that, you can install the packages.

[nur]: https://github.com/goreleaser/nur

```nix
{ pkgs, lib, ... }: {
  home.packages = with pkgs; [
    nur.repos.goreleaser.goreleaser-pro
  ];
}
```

## Docker

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

Download the `.deb`, `.rpm`, or `.apk` packages from the releases page and
install them with the appropriate tools.

{{% g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

To install, after downloading the files, run:

```bash
dpkg -i goreleaser*.deb
rpm -ivh goreleaser*.rpm
apk add --allow-untrusted goreleaser*.apk
```

## Bash Script

This script does not install anything, it just downloads, verifies and runs
GoReleaser.
Its purpose is to be used within scripts and CIs.

```bash
curl -sfL https://goreleaser.com/static/run | DISTRIBUTION=pro VERSION=__VERSION__ bash -s -- check
```

> [!NOTE]
> The `VERSION` environment variable can be omitted to get the latest stable
> version, or you can set it to `nightly` to get the last nightly build.

## Manually

Download the pre-compiled binaries from the releases page and copy them to the
desired location:

{{% g_button href="https://github.com/goreleaser/goreleaser-pro/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

## Verifying the artifacts

### Binaries

#### Signatures

All artifacts are checksummed, and the checksum file is signed with [cosign][].

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

#### Attestations

GitHub does not yet allow cross-repository attestations (e.g. building a
private repo and publishing the attestations in a public one), so this isn't
available yet, unfortunately.

### Docker images

Our Docker images are signed with [cosign][].

Verify the signatures:

```bash
cosign verify \
  --certificate-identity 'https://github.com/goreleaser/goreleaser-pro-internal/.github/workflows/release-pro.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  goreleaser/goreleaser-pro
```

## Nightly builds

Nightly build are pre-releases of the current code into the main branch.
Use it for testing out new features only.

Download the pre-compiled binaries from the
[nightly release](https://github.com/goreleaser/goreleaser-pro/releases/nightly)
and copy them to the desired location.

[Docker](#docker) images are also available, look for tags with a `-nightly`
suffix for the last nightly of a specific release, or the `:nightly` tag,
which is always the latest nightly build available.

You may also use the [Bash Script method](#bash-script) by setting the `VERSION`
environment variable to `nightly`.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/main/Dockerfile
[cosign]: https://github.com/sigstore/cosign
