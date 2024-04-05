# Docker build failures

## `COPY failed: file not found in build context`

This usually happens when trying to build the binary again from source code in
the Docker image build process.

The way GoReleaser works, the correct binary for the platform you're building
should be already available, so you don't need to build it again and can still
reuse the `Dockerfile`.

Another common misconception is trying to copy the binary as if the context is
the repository root.
It's not.
It's always a new temporary build context with the artifacts you can use in
its root, so you can just `COPY binaryname /bin/binaryname` and etc.

Below you can find some **don'ts** as well as what you should **do**.

## `expected to find X artifacts for ids [id1 id2], found Y`

The `ids` property in the Dockers configuration tells GoReleaser which build IDs
to include.
You need to remove IDs that don't exist and/or don't build for the architecture
of the image being built.
Leaving it empty is also fine if you don't need any binaries.

## `use docker --context=default buildx to switch to context "default"`

The "default" context is a built-in context in "docker buildx", and it is automatically created. This context typically points to the local Docker environment and is used by default for building images. It has to be active for `goreleaser` to build images with "buildx".

You can switch to the default context using `docker context use default`.

This change should be persistent.

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

### Do

Copy the clean file names from the root.

```dockerfile
FROM scratch
COPY app /app
ENTRYPOINT ["/app"]
```

!!! tip

    If you still want your users to be able to `docker build` without an extra
    step, you can have a `Dockerfile` just for GoReleaser, for example, a
    `goreleaser.dockerfile`.
