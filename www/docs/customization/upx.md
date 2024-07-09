# UPX

Having small binary sizes are important, and Go is known for generating rather
big binaries.

GoReleaser has had `-s -w` as default `ldflags` since the beginning, which help
shaving off some bytes, but if you want to shave it even more, [`upx`][upx] is
the _de facto_ tool for the job.

GoReleaser has been able to integrate with it via custom [build hooks][bhooks],
and now UPX has its own configuration section:

!!! warning "Compatibility"

    `upx` does not support all platforms! Make sure to check
    [their issues][upx-issues] and to test your packed binaries.

    Namely, _macOS Ventura_ is not supported at the moment.

```yaml
# .goreleaser.yaml
upx:
  - # Whether to enable it or not.
    #
    # Templates: allowed.
    enabled: true

    # Filter by build ID.
    ids: [build1, build2]

    # Filter by GOOS.
    goos: [linux, darwin]

    # Filter by GOARCH.
    goarch: [arm, amd64]

    # Filter by GOARM.
    goarm: [8]

    # Filter by GOAMD64.
    goamd64: [v1]

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

Notice you can define multiple `upx` definitions, filtering by various fields.
You can use that to have different compression options depending on the target
OS, for instance - or even to run it only on a few selected platforms.

{% include-markdown "../includes/templates.md" comments=false %}

[upx]: https://upx.github.io/
[upx-issues]: https://github.com/upx/upx/issues
[bhooks]: builds.md#build-hooks
