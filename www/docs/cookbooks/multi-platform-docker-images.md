# Multi-platform Docker images

On GoReleaser there are two main ways of doing that: the easier one is to use
the [ko integration][ko].

[ko]: ../customization/ko.md

If you don't want to, or can't, use Ko for whatever reason, this guide is for
you!

## Creating Multi-platform docker images with GoReleaser

GoReleaser splits the build and publish phase, which makes its usage less
obvious.

First, you need to define one `dockers` item for each platform you want to
build. Usually, you would tag it like `myorg/myimage:version-platform`.
It is also important to use `buildx`. Here's an example:

```yaml
# .goreleaser.yaml
dockers:
  - image_templates:
      - "myorg/myuser:{{ .Tag }}-amd64"
    use: buildx
    build_flag_templates:
      - "--pull"
      - "--platform=linux/amd64"
  - image_templates:
      - "myorg/myuser:{{ .Tag }}-arm64"
    use: buildx
    build_flag_templates:
      - "--pull"
      - "--platform=linux/arm64"
    goarch: arm64
```

This will, on build time, create two Docker images (`myorg/myuser:v1.2.3-amd64`
and `myorg/myuser:v1.2.3-arm64`).

Now, if we want to make them both available as a single image
(`myorg/myuser:v1.2.3`), we'll need to add a manifest configuration that will
publish them behind that single name. Here's how it would look like:

```yaml
# .goreleaser.yaml
docker_manifests:
  - name_template: "myorg/myuser:{{ .Tag }}"
    image_templates:
      - "myorg/myuser:{{ .Tag }}-amd64"
      - "myorg/myuser:{{ .Tag }}-arm64"
```

And that is it!

## Other things to pay attention to

For `buildx` to work properly, you'll need to install `qemu`. On GitHub actions,
the easiest way is to use
[docker/setup-qemu-action](https://github.com/docker/setup-qemu-action).

It's also important that the `FROM` in your `Dockerfile` is multi-platform,
otherwise it'll not work.

As long as you have Qemu and Docker set up, everything should just work.
