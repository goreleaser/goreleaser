# goreleaser build

Builds the current project

## Synopsis

The `goreleaser build` command is analogous to the `go build` command, in the sense it only builds binaries.

Its intended usage is, for example, within Makefiles to avoid setting up ldflags and etc in several places. That way, the GoReleaser config becomes the source of truth for how the binaries should be built.

It also allows you to generate a local build for your current machine only using the `--single-target` option, and specific build IDs using the `--id` option in case you have more than one.

When using `--single-target`, the `GOOS` and `GOARCH` environment variables are used to determine the target, defaulting to the current machine target if not set.


```
goreleaser build [flags]
```

## Options

```
  -f, --config string      Load configuration from file
  -h, --help               help for build
      --id stringArray     Builds only the specified build ids
  -o, --output string      Copy the binary to the path after the build. Only taken into account when using --single-target and a single id (either with --id or if configuration only has one build)
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --clean            Remove the dist folder before building
      --single-target      Builds only for current GOOS and GOARCH, regardless of what's set in the configuration file
      --skip-after         Skips global after hooks
      --skip-before        Skips global before hooks
      --skip-post-hooks    Skips all post-build hooks
      --skip-validate      Skips several sanity checks
      --snapshot           Generate an unversioned snapshot build, skipping all validations
      --timeout duration   Timeout to the entire build process (default 30m0s)
```

## Options inherited from parent commands

```
      --debug   Enable debug mode
```

## See also

* [goreleaser](/cmd/goreleaser/)	 - Deliver Go binaries as fast and easily as possible

