---
title: Docker Manifest
---

Since [v0.148.0](https://github.com/goreleaser/goreleaser/releases/tag/v0.148.0),
GoReleaser supports building and pushing Docker multi-platform images through
the `docker manifest` tool.

For it to work, it [has to be enabled in the client configurations](https://github.com/docker/cli/blob/master/experimental/README.md).

Please make sure `docker manifest` works before opening issues.

Notice that if you have something in the `docker_manifests` section in your
config file, GoReleaser will add the manifest's to the release notes
instead of the Docker images names.

!!! warning
    Please note that this is a beta feature, and it may change or be removed
    at any time.

## Customization

You can create several manifests in a single GoReleaser run, here are all the
options available:

```yaml
# .goreleaser.yml
docker_manifests:
  # You can have multiple Docker manifests.
-
  # Name template for the manifest.
  # Defaults to empty.
  name_template: foo/bar:{{ .Version }}

  # Image name templates to be added to this manifest.
  # Defaults to empty.
  image_templates:
  - foo/bar:{{ .Version }}-amd64
  - foo/bar:{{ .Version }}-arm64v8

  # Extra flags to be passed down to the manifest create command.
  # Defaults to empty.
  create_flags:
  - --insecure

  # Extra flags to be passed down to the manifest push command.
  # Defaults to empty.
  push_flags:
  - --insecure
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## How it works

We basically build and push our images as usual, but we also add a new
section to our config defining which images are part of which manifests.

GoReleaser will create and publish the manifest in its publish phase.

!!! warning
    Unfortunately, the manifest tool needs the images to be pushed to create
    the manifest, that's why we both create and push it in the publish phase.

## Example config

In this example we will use Docker's `--build-arg` passing an `ARCH` argument.
This way we can use the same `Dockerfile` for both the `amd64` and the `arm64`
images:

```dockerfile
# Dockerfile
ARG ARCH
FROM ${ARCH}/alpine
COPY mybin /usr/bin/mybin
ENTRYPOINT ["/usr/bin/mybin"]
```

Then, on our GoReleaser config file, we need to define both the `dockers` and
the `docker_manifests` section:

```yaml
# .goreleaser.yml
builds:
- env:
  - CGO_ENABLED=0
  binary: mybin
  goos:
  - linux
  goarch:
  - amd64
  - arm64
dockers:
- image_templates:
  - "foo/bar:{{ .Version }}-amd64"
  binaries:
  - mybin
  dockerfile: Dockerfile
  build_flag_templates:
  - "--build-arg"
  - "ARCH=amd64"
- image_templates:
  - "foo/bar:{{ .Version }}-arm64v8"
  binaries:
  - mybin
  goarch: arm64
  dockerfile: Dockerfile
  build_flag_templates:
  - "--build-arg"
  - "ARCH=arm64v8"
docker_manifests:
- name_template: foo/bar:{{ .Version }}
  image_templates:
  - foo/bar:{{ .Version }}-amd64
  - foo/bar:{{ .Version }}-arm64v8
```

!!! warning
    Notice that `ARCH` needs to be in the Docker arch format, not Go's.

That config will build the 2 Docker images defined, as well as the manifest,
and push everything to Docker Hub.
