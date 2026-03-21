---
title: "goreleaser"
weight: 10
---

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

- [goreleaser announce](/cmd/goreleaser_announce/) - Announces a previously prepared release
- [goreleaser build](/cmd/goreleaser_build/) - Builds the current project
- [goreleaser changelog](/cmd/goreleaser_changelog/) - Preview your changelog
- [goreleaser check](/cmd/goreleaser_check/) - Checks if configuration is valid
- [goreleaser completion](/cmd/goreleaser_completion/) - Generate the autocompletion script for the specified shell
- [goreleaser continue](/cmd/goreleaser_continue/) - Continues a previously prepared release
- [goreleaser healthcheck](/cmd/goreleaser_healthcheck/) - Checks if needed tools are installed
- [goreleaser init](/cmd/goreleaser_init/) - Generates a .goreleaser.yaml file
- [goreleaser jsonschema](/cmd/goreleaser_jsonschema/) - Outputs goreleaser's JSON schema
- [goreleaser license-export](/cmd/goreleaser_license-export/) - Export an offline license file from a valid license key
- [goreleaser license-verify](/cmd/goreleaser_license-verify/) - Verify if the given license is valid
- [goreleaser publish](/cmd/goreleaser_publish/) - Publishes a previously prepared release
- [goreleaser release](/cmd/goreleaser_release/) - Releases the current project
- [goreleaser subscribe](/cmd/goreleaser_subscribe/) - Subscribe to GoReleaser Pro, or manage your subscription
