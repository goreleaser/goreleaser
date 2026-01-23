# Docker (v2)

<!-- md:version v2.12 -->

<!-- md:experimental https://github.com/orgs/goreleaser/discussions/6005 -->

!!! warning "alpha"

    This feature is in *alpha* state.
    It will be improved until its deemed stable, at which point we'll
    officially deprecate `dockers` and `docker_manifests` in preparations for
    GoReleaser v3, which should take over both of them.

This feature uses `docker buildx` to build multi-arch manifests,
reusing the previously built binaries and/or packages.

## Customization

Here's a commented out configuration:

```yaml title=".goreleaser.yaml"
dockers_v2:
  # You can have multiple Docker images.
  - #
    # ID of the image, needed if you want to filter by it later on (e.g. on custom publishers).
    # Default: project name
    id: myimg

    # Path to the Dockerfile (from the project root).
    #
    # Default: 'Dockerfile'.
    # Templates: allowed.
    dockerfile: "{{ .Env.DOCKERFILE }}"

    # IDs to filter the binaries/packages.
    #
    # Make sure to only include the IDs of binaries you want to `COPY` in your
    # Dockerfile.
    #
    # If you include IDs that don't exist or are not available for the current
    # architecture being built, the build of the image will be skipped.
    ids:
      - mybuild
      - mynfpm

    # Image names.
    #
    # Empty image names are ignored.
    #
    # Templates: allowed.
    images:
      - "myuser/myimage"
      - "gcr.io/myuser/myimage"

    # Tag names.
    #
    # Empty tags are ignored.
    #
    # Templates: allowed.
    tags:
      - "v{{ .Version }}"
      - "{{ if .IsNightly }}nightly{{ end }}"
      - "{{ if not .IsNightly }}latest{{ end }}"

    # If your Dockerfile copies files other than binaries and packages,
    # you should list them here as well.
    # Note that GoReleaser will create the same structure inside a temporary
    # directory, so if you add `foo/bar.json` here, on your Dockerfile you can
    # `COPY foo/bar.json /whatever.json`.
    # Also note that the paths here are relative to the directory in which
    # GoReleaser is being run (usually the repository root directory).
    # This field does not support wildcards, you can add an entire directory here
    # and use wildcards when you `COPY`/`ADD` in your Dockerfile.
    extra_files:
      - config.yml

    # Labels to be added to the image.
    #
    # Items with empty keys or values will be ignored.
    #
    # Templates: allowed.
    labels:
      "org.opencontainers.image.description": "My software"
      "org.opencontainers.image.created": "{{.Date}}"
      "org.opencontainers.image.name": "{{.ProjectName}}"
      "org.opencontainers.image.revision": "{{.FullCommit}}"
      "org.opencontainers.image.version": "{{.Version}}"
      "org.opencontainers.image.source": "{{.GitURL}}"

    # Annotations to be added to the image.
    #
    # Items with empty keys or values will be ignored.
    #
    # Templates: allowed.
    annotations:
      "foo": "bar"
      "project": "{{.ProjectName}}"

    # Platforms to build.
    #
    # Default: [ linux/amd64 linux/arm64 ]
    # Templates: allowed. (since v2.14)
    platforms:
      - linux/amd64
      - linux/arm64

    # Whether to disable this particular Docker configuration.
    #
    # Templates: allowed.
    # <!-- md:inline_version v2.12.7 -->.
    disable: "{{ .IsSnapshot }}"

    # Whether to create and attach a SBOM to the image.
    #
    # Default: 'true'
    # Templates: allowed.
    # <!-- md:inline_version v2.12.7 -->.
    sbom: "{{ not .IsNightly }}"

    # Additional `--build-arg`s to be passed.
    #
    # Templates: allowed.
    build_args:
      FOO: bar

    # Arbitrary flags to pass to the build command.
    #
    # Note: use this at your own risk.
    # Note: flags must have the `=` sign between flag name and value.
    #
    # Templates: allowed.
    flags:
      - "--ulimit=10"

    # Retry configuration.
    retry:
      # Attempts of retry.
      #
      # Default: 10.
      attempts: 5

      # Delay between retry attempts.
      #
      # Default: 10s.
      delay: 5s

      # Maximum delay between retry attempts.
      #
      # Default: 5m.
      max_delay: 2m
```

!!! important "dockers_v2"

    The `dockers_v2` name is provisional.

    It will replace `dockers` and `docker_manifests` in GoReleaser v3 (no ETA),
    and will then be simply `dockers`.

    We are doing it this way to prevent breaking changes releases now, so we can
    test this new version for a while, before launching v3.

<!-- md:templates -->

## Testing locally

Docker buildx won't allow us to build a manifest without pushing it.
To get around this, when we build with `--snapshot`, GoReleaser will not build
the manifest anymore, and will instead build separated images, adding a platform
suffix to each tag.

Let's see what this means in practice.
Assume we have a configuration like this:

```yaml title=".goreleaser.yaml"
snapshot:
  version_template: "{{ incpatch .Version }}"
dockers_v2:
  - images:
      - user/repo
    tags:
      - "{{.Version}}"
    platforms:
      - linux/amd64
      - linux/arm64
```

If we run `goreleaser release`, i.e., a production build, it'll build and
publish `user/repo:1.2.3`, for example.

If we run `goreleaser release --snapshot`, it'll build two images instead:
`user/repo:1.2.4-amd64` and `user/repo:1.2.4-arm64`.

!!! tip "Daemonless clients"

    If no Docker daemon is detected (e.g., when using remote Buildkit drivers
    like `kubernetes` on daemonless clients in CI env), `goreleaser release --snapshot`
    will automatically skip the `--load` option and build a single multi-arch image
    `user/repo:1.2.4` (similar to `goreleaser release`).

This way you can verify that your Docker build and Docker image work as
expected.

## How it works

You can declare multiple Docker images.
They will be matched against the binaries generated by your `builds` section and
packages generated by your `nfpms` section.

If you have only one item in the `builds` list,
the configuration can be as easy as adding the
name and tags of your images to your `.goreleaser.yaml` file:

```yaml title=".goreleaser.yaml"
dockers_v2:
  - images:
      - user/repo
```

You also need to create a `Dockerfile` in your project's root directory:

```dockerfile title="Dockerfile"
FROM scratch
ARG TARGETPLATFORM
ENTRYPOINT ["/usr/bin/myprogram"]
COPY $TARGETPLATFORM/myprogram /usr/bin/
```

This configuration will build and push a Docker image named `user/repo:tagname`.

### The Docker build context

!!! warning "Don't build binaries in your Dockerfile"

    GoReleaser already builds your binaries (for all target platforms), so you
    don't need to build them again inside the Dockerfile.

    If your Dockerfile has a multi-stage build with a `builder` stage, or
    contains commands like `go build`, `cargo build`, `npm run build`, etc.,
    you're likely duplicating work and **slowing down your builds significantly**.

    Instead, simply copy the pre-built binaries:

    ```dockerfile
    FROM scratch
    ARG TARGETPLATFORM
    ENTRYPOINT ["/usr/bin/myprogram"]
    COPY $TARGETPLATFORM/myprogram /usr/bin/
    ```

    GoReleaser will warn you if it detects patterns that suggest unnecessary
    rebuilds in your `extra_files`.

Note that we are not building any binaries in the `Dockerfile`, we are instead
merely copying the binary to a `scratch` image and setting up the `entrypoint`.

The idea is that you reuse the previously built binaries instead of building
them again when creating the Docker image.

The build context itself is a temporary directory which contains the
binaries and packages for the each of the defined target platforms.
You can then `COPY` them into your image (mind the use of `$TARGETPLATFORM`
above).

A corollary of it being a temporary directory is that
**the context does not contain the source files**.
If you need to add some other file that is in your source directory, you'll
need to add it to the `extra_files` property, so it'll get copied into the
context.

All that being said, your Docker build context will usually look like this:

```sh
temp-context-dir
├── Dockerfile
├── linux/arm64/myprogram
├── linux/arm64/myprogram.rpm
├── linux/arm64/myprogram.apk
├── linux/arm64/myprogram.deb
├── linux/amd64/myprogram
├── linux/amd64/myprogram.rpm
├── linux/amd64/myprogram.apk
└── linux/amd64/myprogram.deb
```

`myprogram` would actually be your binary name, and the Linux package names
would follow their respective configuration's names.

## Setting up a builder

For buildx to work, you'll need to have a builder that supports multi-platform
builds set up.

On Linux, you can do it with:

```sh
docker buildx create --name=goreleaser --use
docker run --privileged --rm tonistiigi/binfmt --install all
```

For what it's worth, this feature was built and tested with buildx v0.24.0.
