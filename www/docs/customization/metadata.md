# Metadata

> Since v1.20

GoReleaser creates some metadata files in the `dist` folder before it finishes
running.

These are the options available:

```yaml
# .goreleaser.yaml
#
metadata:
  # Set the modified timestamp on the metadata files.
  #
  # Templates: allowed.
  mod_timestamp: "{{ .CommitTimestamp }}"
```
