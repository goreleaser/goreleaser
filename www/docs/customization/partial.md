# Splitting and Merging builds

GoReleaser can also split and merge builds.

> Since: v1.12.0-pro.

!!! success "GoReleaser Pro"
    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).

This feature can help in some areas:

1. CGO, as you can build each platform in their target OS and merge later;
1. Native packaging and signing for Windows and macOS (more features for this
   will be added soon);
1. Speed up slow builds, by splitting them into multiple workers;

## Usage

You don't really need to set anything up. To get started, run:

```bash
goreleaser release --rm-dist --split
GOOS=darwin goreleaser release --rm-dist --split
GGOOS=windows goreleaser release --rm-dist --split
```

Note that this step will push your Docker images as well.
Docker manifests are not created yet, though.

- In the first example, it'll build for the current `GOOS` (as returned by
`runtime.GOOS`).
- In the second, it'll use the informed `GOOS`. This env will also bleed to
  things like before hooks, so be aware that any `go run` commands ran by
  GoReleaser there might fail.
- The third example uses the informed `GGOOS`, which is used only to filter
  which targets should be build, and does not affect anything else (as the
  second option does).

Those commands will create the needed artifacts for each platform in
`dist/$GOOS`.

You can also specify `GOARCH` and `GGOARCH`, which only take effect if you set
`partial.by` to `target`.

Now, to continue, run:

```bash
goreleaser continue --merge
```

This last step will run some extra things that were not run during the previous
step:

- merge previous contexts and artifacts lists
- pull previously built images
- create the source archive (if enabled)
- checksum all artifacts
- sign artifacts (according to configuration)
- SBOM artifacts (according to configuration)
- run all the publishers
- run all the announcers

You can also run the publishing and announce steps separately:

```bash
goreleaser publish --merge
goreleaser announce --merge
```

## Customization

You can choose by what you want your pipeline to be split by:

```yaml
# goreleaser.yaml
partial:
  # By what you want to build the partial things.
  #
  # Valid options are `target` and `goos`:
  # - `target`: `GOOS` + `GOARCH`.
  # - `goos`: `GOOS` only
  #
  # Default: `goos`.
  by: target
```

## Integration with GitHub Actions

You can find an example project
[here](https://github.com/caarlos0/goreleaser-pro-split-merge-example).
Feel free to dive into the workflow and the GoReleaser config.
