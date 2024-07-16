# goreleaser announce

Announces a previously prepared release

## Synopsis

If you have a previously prepared release with `goreleaser release --prepare` or `goreleaser release --split` and published it with `goreleaser publish`, you can use this command to announce it.

The idea is to prepare a release without publishing anything, assert the
artifacts are correct (either automatically or not), publish it, and then,
finally, announce it to your users.

Environment variables will be re-evaluated here, so make sure they are
available to the announce command as well.

On the other hand, the GoReleaser configuration file will not be parsed again,
which means you might need to specify the dist directory path if it is different
than the default.

!!! success "GoReleaser Pro"
    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).


```
goreleaser announce [flags]
```

## Options

```
  -d, --dist string        dist directory to continue (default "./dist")
  -h, --help               help for announce
  -k, --key string         GoReleaser Pro license key [$GORELEASER_KEY]
      --merge              Merges multiple parts of a --split release
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --skip strings       Skip the given options (valid options are: after)
      --timeout duration   Timeout to the entire announce process (default 30m0s)
```

## Options inherited from parent commands

```
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](goreleaser.md)	 - Deliver Go binaries as fast and easily as possible

