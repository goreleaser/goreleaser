# Docker build failures

## `COPY failed: file not found in build context`

This usually happens when trying to build the binary again from source code in
the Docker image build process.

The way GoReleaser works, the correct binary for the platform you're building
should be already available, so you don't need to build it again and can still
reuse the `Dockefile`.

Another common misconception is trying to copy the binary as if the context is
the repository root.
It's not.
It's always a new temporary build context with the artifacts you can use in
its root, so you can just `COPY binaryname /bin/binaryname` and etc.

Bellow you can find some **don'ts** as well as what you should **do**.

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

Copy from the `dist` folder.

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
    `goreleaser.dockefile`.
