# goreleaser license-verify

Verify if the given license is valid

## Synopsis

Verifies if the given license is valid.

Will also show some information, like who is the key registered to, which plan it is in, and might also show the expiration time.
		

```
goreleaser license-verify [flags]
```

## Examples

```
# Verify from arguments:
goreleaser license-verify -k "my license"

# Verify from environment variable:
goreleaser license-verify
		
```

## Options

```
  -h, --help         help for license-verify
  -k, --key string   GoReleaser Pro license key [$GORELEASER_KEY] (Pro only)
```

## Options inherited from parent commands

```
      --verbose   Enable verbose mode
```

## See also

* [goreleaser](goreleaser.md)	 - Release engineering, simplified

