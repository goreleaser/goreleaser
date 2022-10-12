# Splitting and Merging builds

!!! success "GoReleaser Pro"
    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).

> Since GoReleaser v1.12.0.

You can now split and merge builds.
This can help in several areas:

1. CGO, as you can build each platform in their target OS and merge later;
1. Native packaging and signing for Windows and macOS (more features for this
   will be added soon);
1. Speed up slow builds, by splitting them into multiple workers;

## Usage

You don't really need to set anything up. To get started, run:

```bash
goreleaser release --rm-dist --split
```

Note that this step will push your Docker images as well.
Docker manifests are not created yet, though.

This will build only the artifacts for your current `GOOS`, and you should be
able to find them in `dist/$GOOS`.

You can run the other `GOOS` you want as well by either running in their OS, or
by giving a `GOOS` to `goreleaser`.

You should then have multiple `GOOS` folder inside your `dist` folder.

Now, to continue, run:

```bash
goreleaser continue
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

