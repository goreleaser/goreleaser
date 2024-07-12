# Docker Images with Ko

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
  - # ID of this image.
    id: foo

    # Build ID that should be used to import the build settings.
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
    # Default: 'cgr.dev/chainguard/static'.
    base_image: alpine

    # Labels for the image.
    labels:
      foo: bar

    # Repository to push to.
    #
    # Default: '$KO_DOCKER_REPO'.
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
    # Valid options are: spdx, cyclonedx, go.version-m and none.
    sbom: none

    # Ldflags to use on build.
    #
    # Default: build.ldflags.
    ldflags:
      - foo
      - bar

    # Flags to use on build.
    #
    # Default: build.flags.
    flags:
      - foo
      - bar

    # Env to use on build.
    #
    # Default: build.env.
    env:
      - FOO=bar
      - SOMETHING=value

    # Bare uses a tag on the $KO_DOCKER_REPO without anything additional.
    bare: true

    # Whether to preserve the full import path after the repository name.
    preserve_import_paths: true

    # Whether to use the base path without the MD5 hash after the repository name.
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
  - env: ["CGO_ENABLED=1"]
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

# Signing KO manifests

KO will add the built manifest to the artifact list, so you can sign them with
`docker_signs`:

```yaml
# .goreleaser.yml
docker_signs:
  - artifacts: manifests
```

[ko]: https://ko.build
[build]: builds.md
