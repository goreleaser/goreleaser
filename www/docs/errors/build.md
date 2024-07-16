# Build Errors

## Undefined methods

If you see an error that looks like this:

```shell
  ⨯ release failed after 14s                 error=failed to build for darwin_amd64_v1: exit status 2: # github.com/rjeczalik/notify
../../../../go/pkg/mod/github.com/rjeczalik/notify@v0.9.2/watcher_fsevents.go:49:11: undefined: stream
../../../../go/pkg/mod/github.com/rjeczalik/notify@v0.9.2/watcher_fsevents.go:200:13: undefined: newStream
```

It usually means that some dependency you are using needs CGO, or does not have
an implementation for the given OS.

You can check that locally with:

```bash
GOOS=darwin GOARCH=amd64 go build ./...
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build ./...
```

If the first fails, but the seconds succeeds, you need to set up
[CGO](../limitations/cgo.md). If both fail, your dependency don't have an
implementation for some methods for Darwin amd64 (in this example).
