## goreleaser build

Builds the current project

```
goreleaser build [flags]
```

### Options

```
  -f, --config string      Load configuration from file
  -h, --help               help for build
      --id string          Builds only the specified build id
  -p, --parallelism int    Amount tasks to run concurrently (default 8)
      --rm-dist            Remove the dist folder before building
      --single-target      Builds only for current GOOS and GOARCH
      --skip-post-hooks    Skips all post-build hooks
      --skip-validate      Skips several sanity checks
      --snapshot           Generate an unversioned snapshot build, skipping all validations and without publishing any artifacts
      --timeout duration   Timeout to the entire build process (default 30m0s)
```

### Options inherited from parent commands

```
      --debug   Enable debug mode
```

### SEE ALSO

* [goreleaser](goreleaser.md)	 - Deliver Go binaries as fast and easily as possible

