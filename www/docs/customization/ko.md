# Docker Images with Ko

> Since v1.14.0.

You can also use [ko][] to build and publish Docker container images.

Please notice that Ko will build your binary again.
That shouldn't increase the release times too much, as it'll use the same build
options as the [build][] pipe when possible, so the results will probably be cached.

!!! warning
    Ko only runs on the publish phase, so it might be a bit hard to test — you
    might need to push to a fake repository (or a fake tag) when playing around
    with its configuration.

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

Refer to the [Ko Build][ko] project page for more information.

[ko]: https://github.com/ko-build/ko
