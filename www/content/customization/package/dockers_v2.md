---
title: "Docker (v2)"
weight: 130
---

{{< g_version "v2.12" >}}

This feature uses `docker buildx` to build multi-arch manifests,
reusing the previously built binaries and/or packages.

## Customization

Here's a commented out configuration:

```yaml {filename=".goreleaser.yaml"}
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

    # Path to a Dockerfile that should be templated before being used.
    #
    # The file contents are rendered as a template, and the result is used as
    # the Dockerfile for the build.
    # When set, it takes precedence over `dockerfile`.
    #
    # When rendering the file contents, `.Binary` (the name of the binary being
    # copied into the image) and `.Binaries` (the sorted list of all binary
    # names, useful when copying more than one) are available - handy for things
    # like `ENTRYPOINT ["/usr/bin/{{ .Binary }}"]`. Since a single image is built
    # for all its platforms, per-platform fields (such as `.Os` and `.Arch`) are
    # not available.
    #
    # Templates: allowed (both the path and the file contents).
    # {{< g_inline_pro >}}
    # {{< g_inline_version "v2.17" >}}
    templated_dockerfile: "Dockerfile.tmpl"

    # IDs filter the binaries and packages copied into the build context.
    # A missing match does not skip the image build.
    # The Dockerfile must work with the selected files for each platform.
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
    # Default: '{{.Tag}}'.
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

    # Same as `extra_files`, but the source files are rendered as templates
    # before being copied into the build context.
    #
    # As with `templated_dockerfile`, `.Binary` and `.Binaries` are available
    # when rendering the file contents.
    #
    # Templates: allowed (source path, destination path, and file contents).
    # {{< g_inline_pro >}}
    # {{< g_inline_version "v2.17" >}}
    templated_extra_files:
      - # Source file path (relative to the project root).
        #
        # Templates: allowed.
        src: config.yml.tmpl

        # Destination path inside the build context.
        #
        # Templates: allowed.
        dst: config.yml

        # File mode.
        #
        # Default: 0o644.
        mode: 0o644

    # Labels to be added to the image.
    #
    # Items with empty keys or values will be ignored.
    #
    # Templates: allowed.
    labels:
      "foo": "bar"
      "project": "{{.ProjectName}}"

    # Annotations to be added to the image.
    #
    # Keys may carry a scope prefix, see "Annotation scopes" below.
    #
    # Items with empty keys or values will be ignored.
    #
    # Templates: allowed.
    annotations:
      "org.opencontainers.image.description": "My software"
      "org.opencontainers.image.created": "{{.Date}}"
      "org.opencontainers.image.name": "{{.ProjectName}}"
      "org.opencontainers.image.revision": "{{.FullCommit}}"
      "org.opencontainers.image.version": "{{.Version}}"
      "org.opencontainers.image.source": "{{.GitURL}}"

      # You can also use `.BaseImage` and `.BaseImageDigest`. {{< g_inline_version "v2.16" >}}
      "org.opencontainers.image.base.name": "{{.BaseImage}}"
      "org.opencontainers.image.base.digest": "{{.BaseImageDigest}}"

      # Keys may be scoped to where the annotation lands. {{< g_inline_version "v2.18" >}}
      "index,manifest:org.opencontainers.image.licenses": "MIT"

    # Platforms to build.
    #
    # Default: [ linux/amd64 linux/arm64 ]
    # Templates: allowed. {{< g_inline_version "v2.14" >}}
    platforms:
      - linux/amd64
      - linux/arm64

    # Whether to disable this particular Docker configuration.
    #
    # Templates: allowed.
    # {{< g_inline_version "v2.12" >}}
    disable: "{{ .IsSnapshot }}"

    # Whether to create and attach a SBOM to the image.
    #
    # Default: 'true'
    # Templates: allowed.
    # {{< g_inline_version "v2.12" >}}
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

    # Custom hooks run around the actual `docker buildx build` invocation.
    # Hooks receive the resolved configuration via templates, so they can
    # inspect or operate on the image plan that's about to be built.
    #
    # Available extra template fields:
    #   - .Dockerfile: the resolved path to the Dockerfile.
    #   - .Images:     the compiled `image:tag` list for this build.
    #   - .ContextDir: the temporary build context directory.
    #   - .Digest:     the resulting image digest (post hook only).
    #
    # {{< g_inline_version "v2.16" >}}
    hooks:
      pre:
        - cmd: ./scripts/before-docker.sh
          # Working directory for the command.
          dir: "{{ .ContextDir }}"
          # Only run this hook if the template evaluates to `true`.
          #
          # {{< g_inline_pro >}}
          # {{< g_inline_version "v2.17" >}}
          if: '{{ eq .Runtime.Goarch "amd64" }}'
          # Extra env vars to inject into the hook.
          env:
            - DOCKERFILE={{ .Dockerfile }}
          # Whether to stream the command output to stdout/stderr.
          output: true
      post:
        - cmd: ./scripts/after-docker.sh {{ .Digest }} {{ range .Images }}{{ . }} {{ end }}
          output: true

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

> [!WARNING]
> **dockers_v2**
>
> The `dockers_v2` name is provisional.
>
> It will replace `dockers` and `docker_manifests` in GoReleaser v3 (no ETA),
> and will then be simply `dockers`.
>
> We are doing it this way to prevent breaking changes releases now, so we can
> test this new version for a while, before launching v3.

{{< g_templates >}}

## Building and pushing are a single step

Docker buildx builds and pushes the manifest in a single `docker buildx build
--push` run, as it can't create a multi-platform manifest locally without
pushing it.

Because of that, `dockers_v2` images are built in the **publish** phase, not in
the build phase (which is what `dockers` used to do).

In practice, this means that anything that skips publishing will also not build
your images:

- `goreleaser build`
- `goreleaser release --skip=publish` (as well as `--skip=docker`)
- `goreleaser release --prepare`{{< g_inline_pro >}}
- `goreleaser release --split`{{< g_inline_pro >}}
- `goreleaser release --single-target`{{< g_inline_pro >}}

The images are then built and pushed later, when you run `goreleaser publish`,
`goreleaser continue`, or `goreleaser continue --merge`{{< g_inline_pro >}}.

> [!TIP]
> If you want to build the images without pushing them, e.g. to verify that your
> `Dockerfile` works, run a [snapshot build](#testing-locally).

## Testing locally

Docker buildx won't allow us to build a manifest without pushing it.
To get around this, when we build with `--snapshot`, GoReleaser will not build
the manifest anymore, and will instead build separated images, adding a platform
suffix to each tag.

Let's see what this means in practice.
Assume we have a configuration like this:

```yaml {filename=".goreleaser.yaml"}
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

> [!NOTE]
> **Daemonless clients**
>
> If no Docker daemon is detected (e.g., when using remote Buildkit drivers
> like `kubernetes` on daemonless clients in CI env), `goreleaser release --snapshot`
> will automatically skip the `--load` option and build a single multi-arch image
> `user/repo:1.2.4` (similar to `goreleaser release`).

This way you can verify that your Docker build and Docker image work as
expected.

## How it works

You can declare multiple Docker images.
They will be matched against the binaries generated by your `builds` section and
packages generated by your `nfpms` section.

If you have only one item in the `builds` list,
the configuration can be as easy as adding the
name and tags of your images to your `.goreleaser.yaml` file:

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - images:
      - user/repo
```

You also need to create a `Dockerfile` in your project's root directory:

```dockerfile {filename="Dockerfile"}
FROM scratch
ARG TARGETPLATFORM
ENTRYPOINT ["/usr/bin/myprogram"]
COPY $TARGETPLATFORM/myprogram /usr/bin/
```

This configuration will build and push a Docker image named `user/repo:tagname`.

### The Docker build context

> [!WARNING]
> **Don't build binaries in your Dockerfile**
>
> GoReleaser already builds your binaries (for all target platforms), so you
> don't need to build them again inside the Dockerfile.
>
> If your Dockerfile has a multi-stage build with a `builder` stage, or
> contains commands like `go build`, `cargo build`, `npm run build`, etc.,
> you're likely duplicating work and **slowing down your builds significantly**.
>
> Instead, simply copy the pre-built binaries:
>
> ```dockerfile {filename="Dockerfile"}
> FROM scratch
> ARG TARGETPLATFORM
> ENTRYPOINT ["/usr/bin/myprogram"]
> COPY $TARGETPLATFORM/myprogram /usr/bin/
> ```
>
> GoReleaser will warn you if it detects patterns that suggest unnecessary
> rebuilds in your `extra_files`.

Note that we are not building any binaries in the `Dockerfile`, we are instead
merely copying the binary to a `scratch` image and setting up the `entrypoint`.

The idea is that you reuse the previously built binaries instead of building
them again when creating the Docker image.

The build context itself is a temporary directory which contains the
binaries and packages for each of the defined target platforms.
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

## Annotation scopes

Annotation keys may be prefixed with the scopes `buildx` should apply them to,
using its `[type:]key=value` syntax. {{< g_inline_version "v2.18" >}}

The available scopes are:

- `index`: the image index, i.e. the multi-platform manifest list.
- `manifest`: each per-platform image manifest.
- `index-descriptor` and `manifest-descriptor`: the descriptors that point to
  them.

Scopes may be comma-separated, and each one may be qualified with a platform:

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - annotations:
      # Index only, the default on multi-platform builds.
      "org.opencontainers.image.description": "My software"

      # Index and every per-platform manifest.
      "index,manifest:org.opencontainers.image.revision": "{{.FullCommit}}"

      # The linux/amd64 manifest only.
      "manifest[linux/amd64]:com.example.arch": "amd64"
```

On multi-platform builds, keys without a scope default to `index:`, so that
tools which inspect the tag itself (such as `docker buildx imagetools inspect`)
see them. Note this differs from a plain `docker buildx build`, which annotates
the per-platform manifests instead.

Scope your keys with `manifest` if you need consumers that resolve a tag down to
a single platform, such as `docker pull`, to see the annotations.

Only the scopes above are recognized: any other prefix is part of the key
itself.

See: [Docker Docs: Annotations > Specify annotation level](https://docs.docker.com/build/metadata/annotations/#specify-annotation-level)

## Docker manifests vs Docker images

This will always use `docker buildx`, which, by default, builds Docker
Manifests.

If you are building single-arch images and want Images instead of Manifests, you
can disable SBOMs and add the `--attest=false` to your configuration, for
example:

```yaml {filename=".goreleaser.yaml"}
dockers_v2:
  - images:
      - foo
    tags:
      - latest

    # These are the important bits:
    platforms:
      - linux/amd64
    sbom: false
    flags:
      - "--provenance=false"
```
