# goreleaser release

Releases the current project

```
goreleaser release [flags]
```

## Options

```
      --auto-snapshot                Automatically sets --snapshot if the repository is dirty
      --clean                        Removes the dist folder
  -f, --config string                Load configuration from file
      --fail-fast                    Whether to abort the release publishing on the first error
  -h, --help                         help for release
      --id stringArray               Builds only the specified build ids (implies --skip-publish) (Pro only)
  -k, --key string                   GoReleaser Pro license key [$GORELEASER_KEY] (Pro only)
      --nightly                      Generate a nightly build, publishing artifacts that support it (implies --skip-announce and --skip-validate) (Pro only)
  -p, --parallelism int              Amount tasks to run concurrently (default: number of CPUs)
      --prepare                      Will run the release in such way that it can be published and announced later with goreleaser publish and goreleaser announce (implies --skip-publish, --skip-announce and --skip-after) (Pro only)
      --release-footer string        Load custom release notes footer from a markdown file
      --release-footer-tmpl string   Load custom release notes footer from a templated markdown file (overrides --release-footer)
      --release-header string        Load custom release notes header from a markdown file
      --release-header-tmpl string   Load custom release notes header from a templated markdown file (overrides --release-header)
      --release-notes string         Load custom release notes from a markdown file (will skip GoReleaser changelog generation)
      --release-notes-tmpl string    Load custom release notes from a templated markdown file (overrides --release-notes)
      --single-target                Builds only for current GOOS and GOARCH, regardless of what's set in the configuration file (implies --skip-publish) (Pro only)
      --skip-after                   Skips global after hooks (Pro only)
      --skip-announce                Skips announcing releases (implies --skip-validate)
      --skip-before                  Skips global before hooks
      --skip-docker                  Skips Docker Images/Manifests builds
      --skip-fury                    Skips Fury publishing (Pro only)
      --skip-ko                      Skips Ko builds
      --skip-publish                 Skips publishing artifacts (implies --skip-announce)
      --skip-sbom                    Skips cataloging artifacts
      --skip-sign                    Skips signing artifacts
      --skip-validate                Skips git checks
      --snapshot                     Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts (implies --skip-publish, --skip-announce and --skip-validate, overrides --nightly)
      --split                        Split the build so it can be merged and published later (implies --prepare) (Pro only)
      --timeout duration             Timeout to the entire release process (default 30m0s)
```

## Options inherited from parent commands

```
      --debug     Enable verbose mode (deprecated)
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](/cmd/goreleaser/)	 - Deliver Go binaries as fast and easily as possible

