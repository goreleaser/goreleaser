# goreleaser release

Releases the current project

```
goreleaser release [flags]
```

## Options

```
      --auto-snapshot                Automatically sets --snapshot if the repo is dirty
  -f, --config string                Load configuration from file
  -h, --help                         help for release
  -k, --key string                   GoReleaser Pro license key [$GORELEASER_KEY]
      --nightly                      Generate a nightly build, publishing artifacts that support it (implies --skip-announce and --skip-validate)
  -p, --parallelism int              Amount tasks to run concurrently (default: number of CPUs)
      --release-footer string        Load custom release notes footer from a markdown file
      --release-footer-tmpl string   Load custom release notes footer from a templated markdown file (overrides --release-footer)
      --release-header string        Load custom release notes header from a markdown file
      --release-header-tmpl string   Load custom release notes header from a templated markdown file (overrides --release-header)
      --release-notes string         Load custom release notes from a markdown file
      --release-notes-tmpl string    Load custom release notes from a templated markdown file (overrides --release-notes)
      --rm-dist                      Removes the dist folder
      --skip-announce                Skips announcing releases (implies --skip-validate)
      --skip-publish                 Skips publishing artifacts
      --skip-sbom                    Skips cataloging artifacts
      --skip-sign                    Skips signing artifacts
      --skip-validate                Skips git checks
      --snapshot                     Generate an unversioned snapshot release, skipping all validations and without publishing any artifacts (implies --skip-publish, --skip-announce and --skip-validate, overrides --nightly)
      --timeout duration             Timeout to the entire release process (default 30m0s)
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser](/cmd/goreleaser/)	 - Deliver Go binaries as fast and easily as possible

