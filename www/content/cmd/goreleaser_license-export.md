# goreleaser license-export

Export an offline license file from a valid license key

## Synopsis

Allows to export an offline license file from a valid license key.

These keys have an expiration time of at most 90 days, and are signed by our server.

This is only available for customers for the plans 'business' and 'enterprise'

```
goreleaser license-export [flags]
```

## Examples

```

# Read from arguments and xport to a file:
goreleaser license-export -k "my license" -o goreleaser.key

# Read from environment variable and export to STDOUT:
goreleaser license-export -o -
		
```

## Options

```
  -h, --help            help for license-export
  -k, --key string      GoReleaser Pro license key [$GORELEASER_KEY] (Pro only)
  -o, --output string   Output file path (default "-")
```

## Options inherited from parent commands

```
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](goreleaser.md)	 - Release engineering, simplified

