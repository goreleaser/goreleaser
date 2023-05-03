# UPX

> Since: v1.18

Having small binary sizes are important, and Go is known for generating rather
big binaries.

GoReleaser has had `-s -w` as default `ldflags` since the beginning, which help
shaving off some bytes, but if you want to shave it even more, [`upx`][upx] is
the _de facto_ tool for the job.

[upx]: https://upx.github.io/

GoReleaser has been able to integrate with it via custom [build hooks][bhooks],
and now UPX has its own configuration section:

```yaml
# .goreleaser.yaml
upx:
  -
    # Whether to enable it or not.
    enabled: true

    # Filter by build ID.
    ids: [ build1, build2 ]

    # Compress argument.
    # Valid options are from '1' (faster) to '9' (better), and 'best'.
    compress: best

    # Whether to try LZMA (slower).
    lzma: true

    # Whether to try all methods and filters (slow).
    brute: true
```

!!! info
    If `upx` is not in `$PATH`, GoReleaser will automatically avoid running it.
