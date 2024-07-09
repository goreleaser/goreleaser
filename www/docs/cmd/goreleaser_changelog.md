# goreleaser changelog

Preview your changelog

## Synopsis

The `goreleaser changelog` command can be used to preview your next release changelog.

It'll get the changes from the latest tag to the current commit, and print them to standard output or to a file.

You can also use this command to test the `changelog` configuration in your `.goreleaser.yml` file.

This command skips all validations and does not publish anything.

!!! success "GoReleaser Pro"
    This subcommand is a [GoReleaser Pro feature](https://goreleaser.com/pro/).


```
goreleaser changelog
```

## Options

```
  -f, --config string      Load configuration from file
  -h, --help               help for changelog
  -o, --output string      File to save the changelog to, if empty prints it to STDOUT
      --timeout duration   Timeout to the entire build process (default 1m0s)
```

## Options inherited from parent commands

```
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](goreleaser.md)	 - Deliver Go binaries as fast and easily as possible

