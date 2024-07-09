# goreleaser publish

Publishes a previously prepared release

## Synopsis

If you have a previously prepared release (run with `goreleaser release --prepare` or `goreleaser release --split`), you can use this command to publish it.

The idea is to prepare a release without publishing anything, assert the
artifacts are correct (either automatically or not), and then, finally, publish
the release and its artifacts.

Environment variables will be re-evaluated here, so make sure they are
available to the publish command as well.

!!! success "GoReleaser Pro"
    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).


```
goreleaser publish [flags]
```

## Options

```
  -d, --dist string        dist directory to continue (default "./dist")
  -h, --help               help for publish
  -k, --key string         GoReleaser Pro license key [$GORELEASER_KEY]
      --merge              Merges multiple parts of a --split release
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --skip strings       Skip the given options (valid options are: after)
      --timeout duration   Timeout to the entire publish process (default 30m0s)
```

## Options inherited from parent commands

```
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](goreleaser.md)	 - Deliver Go binaries as fast and easily as possible

