# UPX

> Since: v1.18

Having small binary sizes are important, and Go is known for generating rather
big binaries.

GoReleaser has had `-s -w` as default `ldflags` since the beginning, which help
shaving off some bytes, but if you want to shave it even more, [`upx`][upx] is
the _de facto_ tool for the job.

GoReleaser has been able to integrate with it via custom [build hooks][bhooks],
and now UPX has its own configuration section:

!!! warning
    `upx` does not support all platforms! Make sure to check
    [their issues][upx-issues] and to test your packed binaries first.

    Namely, _macOS Ventura_ is not supported at the moment.

    Future GoReleaser releases will add more filters so you can cherry-pick
    which platforms you want to pack or not.

```yaml
# .goreleaser.yaml
upx:
  -
    # Whether to enable it or not.
    enabled: true

    # Filter by build ID.
    ids: [ build1, build2 ]

    # Filter by GOOS.
    #
    # Since: v1.19
    goos: [ linux , darwin ]

    # Filter by GOARCH.
    #
    # Since: v1.19
    goarch: [ arm, amd64 ]

    # Filter by GOARM.
    #
    # Since: v1.19
    goarm: [ 8 ]

    # Filter by GOAMD64.
    #
    # Since: v1.19
    goamd64: [ v1 ]

    # Compress argument.
    # Valid options are from '1' (faster) to '9' (better), and 'best'.
    compress: best

    # Whether to try LZMA (slower).
    lzma: true

    # Whether to try all methods and filters (slow).
    brute: true
```

Notice you can define multiple `upx` definitions, filtering by various fields.
You can use that to have different compression options depending on the target
OS, for instance - or even to run it only on a few selected platforms.

!!! info
    If `upx` is not in `$PATH`, GoReleaser will automatically avoid running it.

[upx]: https://upx.github.io/
[upx-issues]: https://github.com/upx/upx/issues
[bhooks]: /customization/builds/#build-hooks
