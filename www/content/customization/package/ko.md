---
title: "Docker Images with Ko"
linkTitle: Ko
weight: 140
---

You can also use [ko][] to build and publish Docker container images.

Please notice that ko will build your binary again.
That shouldn't increase the release times too much, as it'll use the same build
options as the [build][] pipe when possible, so the results will probably be
cached.

> [!WARNING]
> When on `--snapshot` mode, Ko will publish the image to `goreleaser.ko.local`.
> If it's a regular build, Ko will only run in the publishing phase.

> [!NOTE]
> For Ko to work you still need to log in, either with `docker login` or
> something else.

```yaml {filename=".goreleaser.yaml"}
kos:
  - # ID of this image.
    #
    # Default: project name.
    id: foo

    # Build ID that should be used to import the build settings.
    #
    # Default: id.
    build: build-id

    # Main path to build.
    # It must be a relative path
    #
    # Default: build.main.
    main: ./cmd/...

    # Working directory used to build.
    #
    # Default: build.dir.
    working_dir: .

    # Base image to publish to use.
    #
    # Local images will take priority over fetching remote images. {{< g_inline_version "v2.13" >}}
    #
    # Default: 'cgr.dev/chainguard/static'.
    # Templates: allowed.
    base_image: alpine

    # Labels for the image.
    #
    # Templates: allowed (values only).
    labels:
      foo: bar

    # Annotations for the OCI manifest.
    #
    # Templates: allowed (values only).
    annotations:
      foo: bar

    # The default user the image should be run as.
    user: "1234:1234"

    # Repositories to push to.
    #
    # First one will be used on Ko build, the other ones will be copied from the
    # first one using crane.
    #
    # Default: [ '$KO_DOCKER_REPO' ].
    # Templates: allowed.
    repositories:
      - ghcr.io/foo/bar
      - foo/bar

    # Repository to push to.
    #
    # Default: '$KO_DOCKER_REPO'.
    # Deprecated: use 'repositories' instead.
    # Templates: allowed.
    repository: ghcr.io/foo/bar

    # Platforms to build and publish.
    #
    # Default: 'linux/amd64'.
    platforms:
      - linux/amd64
      - linux/arm64

    # Tag to build and push.
    # Empty tags are ignored.
    #
    # Default: 'latest'.
    # Templates: allowed.
    tags:
      - latest
      - "{{.Tag}}"
      - "{{if not .Prerelease}}stable{{end}}"

    # Creation time given to the image
    # in seconds since the Unix epoch as a string.
    #
    # Templates: allowed.
    creation_time: "{{.CommitTimestamp}}"

    # Creation time given to the files in the kodata directory
    # in seconds since the Unix epoch as a string.
    #
    # Templates: allowed.
    ko_data_creation_time: "{{.CommitTimestamp}}"

    # SBOM format to use.
    #
    # Default: 'spdx'.
    # Valid options are: spdx and none.
    sbom: none

    # Path to file where the SBOM will be written
    #
    # Default: unset - no SBOM written to filesystem (but still uploaded to oci repository).
    sbom_directory: "out/sbom"

    # Ko publishes images to the local Docker daemon
    # when GoReleaser is executed with the --snapshot flag.
    # Use the local_domain attribute to configure the local registry (e.g. kind.local).
    #
    # Default "goreleaser.ko.local" - local docker registry is used.
    # Templates: allowed.
    # {{< g_inline_version "v2.10" >}}
    local_domain: "goreleaser.ko.local"

    # Ldflags to use on build.
    #
    # Default: build.ldflags.
    # Templates: allowed.
    ldflags:
      - foo
      - bar

    # Flags to use on build.
    #
    # Default: build.flags.
    # Templates: allowed.
    flags:
      - foo
      - bar

    # Env to use on build.
    #
    # Default: build.env.
    # Templates: allowed.
    env:
      - FOO=bar
      - SOMETHING=value

    # Whether to disable this particular Ko configuration.
    #
    # Templates: allowed.
    # {{< g_inline_version "v2.8" >}}
    disable: "{{ .IsSnapshot }}"

    # Bare uses a tag on the $KO_DOCKER_REPO without anything additional.
    bare: true

    # Whether to preserve the full import path after the repository name.
    preserve_import_paths: true

    # Whether to use the base path without the MD5 hash after the repository name.
    base_import_paths: true
```

Refer to [ko's project page][ko] for more information.

> [!WARNING]
> Note that while GoReleaser's build section will evaluate environment
> variables for each target being built, Ko doesn't.
> This means that variables like `.Os`, `.Arch`, and the sorts, will not be
> available.

## Example

Here's a minimal example:

```yaml {filename=".goreleaser.yaml"}
before:
  hooks:
    - go mod tidy

builds:
  - env: ["CGO_ENABLED=1"]
    binary: test
    goos:
      - darwin
      - linux
    goarch:
      - amd64
      - arm64

kos:
  - repositories: [ghcr.io/caarlos0/test-ko]
    tags:
      - "{{.Version}}"
      - latest
    bare: true
    preserve_import_paths: false
    platforms:
      - linux/amd64
      - linux/arm64
```

This will build the binaries for `linux/arm64`, `linux/amd64`, `darwin/amd64`
and `darwin/arm64`, as well as the Docker images and manifest for Linux.

## Signing KO manifests

KO will add the built manifest to the artifact list, so you can sign them with
`docker_signs`:

```yaml {filename=".goreleaser.yaml"}
docker_signs:
  - artifacts: manifests
```

[ko]: https://ko.build
[build]: /customization/builds/builders/go/
