---
title: Docker build failures
weight: 50
---

## `COPY failed: file not found in build context`

This usually happens when trying to build the binary again from source code in
the Docker image build process.

The way GoReleaser works, the correct binary for the platform you're building
should be already available, so you don't need to build it again and can still
reuse the `Dockerfile`.

Another common misconception is trying to copy the binary as if the context is
the repository root.
It's not.
GoReleaser creates a temporary build context. The artifact paths depend on
which Docker integration you use:

- [Docker v2](/customization/package/dockers_v2/) puts binaries and packages in
  platform directories, such as `linux/amd64/`.
- [Legacy Docker](/customization/package/docker/) puts them at the context root.

Below you can find some **don'ts** as well as what you should **do**.

### Don't

Build the binary again.

```dockerfile
FROM golang AS builder
WORKDIR /app
COPY cmd ./cmd
COPY go.mod ./
COPY *.go ./
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o app .

FROM scratch
COPY --from=builder /app/app /app
ENTRYPOINT ["/app"]
```

### Don't

Copy from the `dist` directory.

```dockerfile
FROM scratch
COPY /dist/app_linux_amd64/app /app
ENTRYPOINT ["/app"]
```

### Do: Docker v2

Use Docker's `TARGETPLATFORM` build argument to select the platform directory:

```dockerfile
FROM scratch
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/app /app
ENTRYPOINT ["/app"]
```

Docker supplies `TARGETPLATFORM`, for example `linux/amd64`.

### Do: legacy Docker

With the legacy `dockers` configuration, copy the binary from the context root:

```dockerfile
FROM scratch
COPY app /app
ENTRYPOINT ["/app"]
```

> [!NOTE]
> If you still want your users to be able to `docker build` without an extra
> step, you can have a `Dockerfile` just for GoReleaser, for example, a
> `goreleaser.dockerfile`.

## `expected to find X artifacts for ids [id1 id2], found Y`

The `ids` property in the Dockers configuration tells GoReleaser which build IDs
to include.
You need to remove IDs that don't exist and/or don't build for the architecture
of the image being built.
Leaving it empty is also fine if you don't need any binaries.

## `use docker --context=default buildx to switch to context "default"`

The `default` context is a built-in context in `docker buildx`, and it is
created automatically. This context typically points to the local Docker
environment and is used by default for building images. It has to be active for
`goreleaser` to build images with `buildx`.

You can switch to the default context using `docker context use default`.

This change should be persistent.
