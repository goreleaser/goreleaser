---
title: "Install GoReleaser"
linkTitle: OSS
weight: 10
description: "Installing the open source GoReleaser distribution."
---

{{< cards cols="2" >}}
{{< card link="https://github.com/goreleaser/goreleaser/releases/tag/__VERSION__" title="Latest stable" subtitle="`__VERSION__`" >}}
{{< card link="https://github.com/goreleaser/goreleaser/releases/nightly" title="Latest nightly" subtitle="<span data-nightly-tag data-repo='goreleaser/goreleaser'>loading…</span>" >}}
{{< /cards >}}

See all releases on [GitHub](https://github.com/goreleaser/goreleaser/releases).

{{< g_install_versions >}}

## Homebrew Tap

```bash
brew install --cask goreleaser/tap/goreleaser
```

## Homebrew

<br>
{{< badge content="Community Owned" icon="external-link" >}}

```bash
brew install goreleaser
```

> [!WARNING]
> The formula in homebrew-core might be slightly outdated.
> Use our homebrew tap to always get the latest updates.

[formula in homebrew-core]: https://github.com/Homebrew/homebrew-core/blob/master/Formula/g/goreleaser.rb

## NPM

```bash
npm i -g @goreleaser/goreleaser
```

## Snapcraft

```bash
sudo snap install --classic goreleaser
```

## Scoop

```bash
scoop bucket add goreleaser https://github.com/goreleaser/scoop-bucket.git
scoop install goreleaser
```

## Chocolatey

<br>
{{< badge content="Community Owned" icon="external-link" >}}

```bash
choco install goreleaser
```

## Winget

```bash
winget install goreleaser
```

## Apt Repository

```bash
echo 'deb [trusted=yes] https://repo.goreleaser.com/apt/ /' | sudo tee /etc/apt/sources.list.d/goreleaser.list
sudo apt update
sudo apt install goreleaser
```

## Yum Repository

```bash
echo '[goreleaser]
name=GoReleaser
baseurl=https://repo.goreleaser.com/yum/
enabled=1
gpgcheck=0
exclude=goreleaser-pro' | sudo tee /etc/yum.repos.d/goreleaser.repo
sudo yum install goreleaser
```

## AUR

```bash
yay -S goreleaser-bin
```

## Nixpkgs

<br>
{{< badge content="Community Owned" icon="external-link" >}}

```bash
nix-shell -p goreleaser
```

> [!WARNING]
> The package in nixpkgs might be slightly outdated, as it is not
> updated automatically.
> Use our NUR to always get the latest updates.

[package in nixpkgs]: https://github.com/NixOS/nixpkgs/blob/master/pkgs/tools/misc/goreleaser/default.nix

## NUR

First, you'll need to add our [NUR][nur] to your nix configuration.
You can follow the guides
[here](https://github.com/nix-community/NUR#installation).

Once you do that, you can install the packages.

[nur]: https://github.com/goreleaser/nur

```nix
{ pkgs, lib, ... }: {
  home.packages = with pkgs; [
    nur.repos.goreleaser.goreleaser
  ];
}
```

## Docker

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

{{% g_button href="https://github.com/goreleaser/goreleaser/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

To install, after downloading the files, run:

```bash
dpkg -i goreleaser*.deb
rpm -ivh goreleaser*.rpm
apk add --allow-untrusted goreleaser*.apk
```

## `go install`

```bash
go install github.com/goreleaser/goreleaser/v2@latest
```

Requires Go 1.26.

## Bash Script

This script does not install anything, it just downloads, verifies and runs
GoReleaser.
Its purpose is to be used within scripts and CIs.

```bash
curl -sfL https://goreleaser.com/static/run | bash VERSION=__VERSION__ -s -- check
```

> [!NOTE]
> The `VERSION` environment variable can be omitted to get the latest stable
> version, or you can set it to `nightly` to get the last nightly build.

## Manually

Download the pre-compiled binaries from the releases page and copy them to the
desired location:

{{% g_button href="https://github.com/goreleaser/goreleaser/releases/tag/__VERSION__" label="Download" icon="github" primary="true" %}}

## Verifying the artifacts

### Binaries

#### Signatures

All artifacts are checksummed, and the checksum file is signed with [cosign][].

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

#### Attestations

You can also verify the attestations:

```bash
gh attestation verify --owner goreleaser *.tar.gz
# PS: can be any file from the release
```

### Docker images

Our Docker images are signed with [cosign][].

Verify the signatures:

```bash
cosign verify \
  --certificate-identity 'https://github.com/goreleaser/goreleaser/.github/workflows/release.yml@refs/tags/__VERSION__' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  goreleaser/goreleaser
```

## Nightly builds

Nightly build are pre-releases of the current code into the main branch.
Use it for testing out new features only.

Download the pre-compiled binaries from the
[nightly release](https://github.com/goreleaser/goreleaser/releases/nightly)
and copy them to the desired location.

[Docker](#docker) images are also available, look for tags with a `-nightly`
suffix for the last nightly of a specific release, or the `:nightly` tag,
which is always the latest nightly build available.

You may also use the [Bash Script method](#bash-script) by setting the `VERSION`
environment variable to `nightly`.

## Community

Install options with the "Community Owned" badge are maintained by the community
and might not always be up to date.

[dockerfile]: https://github.com/goreleaser/goreleaser/blob/main/Dockerfile
[cosign]: https://github.com/sigstore/cosign
