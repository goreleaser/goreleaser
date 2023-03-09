# Docker Images with Ko

> Since v1.15.0.

You can also use [ko][] to build and publish Docker container images.

Please notice that ko will build your binary again.
That shouldn't increase the release times too much, as it'll use the same build
options as the [build][] pipe when possible, so the results will probably be
cached.

!!! warning
    Ko only runs on the publishing phase, so it might be a bit hard to test —
    you might need to push to a fake repository (or a fake tag) when playing
    around with its configuration.

```yaml
# .goreleaser.yaml
kos:
-
  # ID of this image.
  id: foo

  # Build ID that should be used to import the build settings.
  build: build-id

  # Main path to build.
  #
  # Defaults to the build's main.
  main: ./cmd/...

  # Working directory used to build.
  #
  # Defaults to the build's dir.
  working_dir: .

  # Base image to publish to use.
  #
  # Defaults to cgr.dev/chainguard/static.
  base_image: alpine

  # Labels for the image.
  #
  # Defaults to null.
  # Since v1.17.
  labels:
    foo: bar

  # Repository to push to.
  #
  # Defaults to the value of $KO_DOCKER_REPO.
  repository: ghcr.io/foo/bar

  # Platforms to build and publish.
  #
  # Defaults to linux/amd64.
  platforms:
  - linux/amd64
  - linux/arm64

  # Tag templates to build and push.
  #
  # Defaults to `latest`.
  tags:
  - latest
  - '{{.Tag}}'

  # SBOM format to use.
  #
  # Defaults to spdx.
  # Valid options are: spdx, cyclonedx, go.version-m and none.
  sbom: none

  # Ldflags to use on build.
  #
  # Defaults to the build's ldflags.
  ldflags:
  - foo
  - bar

  # Flags to use on build.
  #
  # Defaults to the build's flags.
  flags:
  - foo
  - bar

  # Env to use on build.
  #
  # Defaults to the build's env.
  env:
  - FOO=bar
  - SOMETHING=value


  # Bare uses a tag on the KO_DOCKER_REPO without anything additional.
  #
  # Defaults to false.
  bare: true

  # Whether to preserve the full import path after the repository name.
  #
  # Defaults to false.
  preserve_import_paths: true

  # Whether to use the base path without the MD5 hash after the repository name.
  #
  # Defaults to false.
  base_import_paths: true
```

Refer to [ko's project page][ko] for more information.


## Example

Here's a minimal example:

```yaml
# .goreleaser.yml
before:
  hooks:
    - go mod tidy

builds:
  - env: [ "CGO_ENABLED=1" ]
    binary: test
    goos:
    - darwin
    - linux
    goarch:
    - amd64
    - arch64

kos:
  - repository: ghcr.io/caarlos0/test-ko
    tags:
    - '{{.Version}}'
    - latest
    bare: true
    preserve_import_paths: false
    platforms:
    - linux/amd64
    - linux/arm64
```

This will build the binaries for `linux/arm64`, `linux/amd64`, `darwin/amd64`
and `darwin/arm64`, as well as the Docker images and manifest for Linux.

[ko]: https://ko.build
[build]: /customization/build/
