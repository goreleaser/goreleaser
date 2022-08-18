# goreleaser announce

Announces a previously prepared release

## Synopsis

If you have a previously prepared release with `goreleaser release --prepare` and published it with `goreleaser publish`, you can use this command to announce it.

The idea is to prepare a release without publishing anything, assert the artifacts are correct (either automatically or not), publish it, and then, finally, announce it to your users.

Environment variables will be re-evaluated here, so make sure they are available to the announce command as well.

!!! success "GoReleaser Pro"
    This subcommand is [GoReleaser Pro feature](https://goreleaser.com/pro/).


```
goreleaser announce [flags]
```

## Options

```
  -d, --dist string        dist folder to continue (default "./dist")
  -h, --help               help for announce
  -k, --key string         GoReleaser Pro license key [$GORELEASER_KEY]
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --skip-after         Skips global after hooks
      --timeout duration   Timeout to the entire announce process (default 30m0s)
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser](/cmd/goreleaser/)	 - Deliver Go binaries as fast and easily as possible

