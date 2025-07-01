# goreleaser

Release engineering, simplified

## Synopsis

Release engineering, simplified.

GoReleaser is a release automation tool, built with love and care by @caarlos0 and many contributors.

Complete documentation is available at https://goreleaser.com

## Examples

```

# Initialize your project:
goreleaser init

# Verify your configuration:
goreleaser check

# Verify dependencies:
goreleaser healthcheck

# Build the binaries only:
goreleaser build

# Run a snapshot release:
goreleaser release --snapshot

# Run a complete release:
goreleaser release
		
```

## Options

```
  -h, --help      help for goreleaser
      --verbose   Enable verbose mode
```

## See also

* [goreleaser announce](goreleaser_announce.md)	 - Announces a previously prepared release
* [goreleaser build](goreleaser_build.md)	 - Builds the current project
* [goreleaser changelog](goreleaser_changelog.md)	 - Preview your changelog
* [goreleaser check](goreleaser_check.md)	 - Checks if configuration is valid
* [goreleaser completion](goreleaser_completion.md)	 - Generate the autocompletion script for the specified shell
* [goreleaser continue](goreleaser_continue.md)	 - Continues a previously prepared release
* [goreleaser healthcheck](goreleaser_healthcheck.md)	 - Checks if needed tools are installed
* [goreleaser init](goreleaser_init.md)	 - Generates a .goreleaser.yaml file
* [goreleaser jsonschema](goreleaser_jsonschema.md)	 - Outputs goreleaser's JSON schema
* [goreleaser mcp](goreleaser_mcp.md)	 - Start a MCP server that provides GoReleaser tools
* [goreleaser publish](goreleaser_publish.md)	 - Publishes a previously prepared release
* [goreleaser release](goreleaser_release.md)	 - Releases the current project
* [goreleaser subscribe](goreleaser_subscribe.md)	 - Subscribe to GoReleaser Pro, or manage your subscription
* [goreleaser verify-license](goreleaser_verify-license.md)	 - Verify if the given license is valid

