---
weight: 10
---# goreleaser

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

* [goreleaser announce](goreleaser_announce/)	 - Announces a previously prepared release
* [goreleaser build](goreleaser_build/)	 - Builds the current project
* [goreleaser changelog](goreleaser_changelog/)	 - Preview your changelog
* [goreleaser check](goreleaser_check/)	 - Checks if configuration is valid
* [goreleaser completion](goreleaser_completion/)	 - Generate the autocompletion script for the specified shell
* [goreleaser continue](goreleaser_continue/)	 - Continues a previously prepared release
* [goreleaser healthcheck](goreleaser_healthcheck/)	 - Checks if needed tools are installed
* [goreleaser init](goreleaser_init/)	 - Generates a .goreleaser.yaml file
* [goreleaser jsonschema](goreleaser_jsonschema/)	 - Outputs goreleaser's JSON schema
* [goreleaser license-export](goreleaser_license-export/)	 - Export an offline license file from a valid license key
* [goreleaser license-verify](goreleaser_license-verify/)	 - Verify if the given license is valid
* [goreleaser publish](goreleaser_publish/)	 - Publishes a previously prepared release
* [goreleaser release](goreleaser_release/)	 - Releases the current project
* [goreleaser subscribe](goreleaser_subscribe/)	 - Subscribe to GoReleaser Pro, or manage your subscription

