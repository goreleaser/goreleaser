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
  # ID of the manifest, needed if you want to filter by it later on (e.g. on custom publishers).
  id: myimg

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

  # Skips the Docker manifest.
  # If you set this to 'false' or 'auto' on your source Docker configs,
  #  you'll probably want to do the same here.
  #
  # If set to 'auto', the manifest will not be created in case there is an
  #  indicator of a prerelease in the tag, e.g. v1.0.0-rc1.
  #
  # Defaults to false.
  skip_push: false

  # Set the "backend" for the Docker manifest pipe.
  # Valid options are: docker, podman
  #
  # Relevant notes:
  # 1. podman is a GoReleaser Pro feature and is only available on Linux;
  # 2. if you set podman here, the respective docker configs need to use podman too.
  #
  # Defaults to docker.
  use: docker
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

In this example we will use Docker's `--platform` option to specify the target platform.
This way we can use the same `Dockerfile` for both the `amd64` and the `arm64`
images (and possibly others):

```dockerfile
# Dockerfile
FROM alpine
ENTRYPOINT ["/usr/bin/mybin"]
COPY mybin /usr/bin/mybin
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
  use: buildx
  dockerfile: Dockerfile
  build_flag_templates:
  - "--platform=linux/amd64"
- image_templates:
  - "foo/bar:{{ .Version }}-arm64v8"
  use: buildx
  goarch: arm64
  dockerfile: Dockerfile
  build_flag_templates:
  - "--platform=linux/arm64/v8"
docker_manifests:
- name_template: foo/bar:{{ .Version }}
  image_templates:
  - foo/bar:{{ .Version }}-amd64
  - foo/bar:{{ .Version }}-arm64v8
```

!!! warning
    Notice that `--platform` needs to be in the Docker platform format, not Go's.

That config will build the 2 Docker images defined, as well as the manifest,
and push everything to Docker Hub.

## Podman

You can use [`podman`](https://podman.io) instead of `docker` by setting `use` to `podman` on your config:

```yaml
# .goreleaser.yml
docker_manifests:
- name_template: foo/bar:{{ .Version }}
  image_templates:
  - foo/bar:{{ .Version }}-amd64
  - foo/bar:{{ .Version }}-arm64v8
  use: podman
```

Note that GoReleaser will not install Podman for you, nor change any of its configuration.
Also worth noticing that currently Podman only works on Linux machines.

!!! info
    The Podman backend is a [GoReleaser Pro feature](/pro/).
