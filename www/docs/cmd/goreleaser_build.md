# goreleaser build

Builds the current project

## Synopsis

The build command allows you to execute only a subset of the pipeline, i.e. only the build step with its dependencies.

It allows you to quickly check if your GoReleaser build configurations are doing what you expect.

Finally, it allows you to generate a local build for your current machine only using the `--single-target` option, and specific build IDs using the `--id` option.


```
goreleaser build [flags]
```

## Options

```
  -f, --config string      Load configuration from file
  -h, --help               help for build
      --id string          Builds only the specified build id
  -p, --parallelism int    Amount tasks to run concurrently (default: number of CPUs)
      --rm-dist            Remove the dist folder before building
      --single-target      Builds only for current GOOS and GOARCH
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

