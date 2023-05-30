# goreleaser continue

Continues a previously prepared release

## Synopsis

If you have a previously prepared release (run with `goreleaser release --prepare` or `goreleaser release --split`), you can use this command to continue it.

Environment variables will be re-evaluated here, so make sure they are
available to the continue command as well.

!!! success "GoReleaser Pro"

    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).

```
goreleaser continue [flags]
```

## Options

```
  -d, --dist string        dist folder to continue (default "./dist")
  -h, --help               help for continue
  -k, --key string         GoReleaser Pro license key [$GORELEASER_KEY]
      --merge              Merges multiple parts of a --split release
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --skip-after         Skips global after hooks
      --timeout duration   Timeout to the entire continue process (default 30m0s)
```

## Options inherited from parent commands

```
      --debug     Enable verbose mode (deprecated)
      --verbose   Enable verbose mode
```

## See also

- [goreleaser](/cmd/goreleaser/) - Deliver Go binaries as fast and easily as possible
