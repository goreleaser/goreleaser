# Metadata

> Since v1.20

Here's the available configuration options:

```yaml
# .goreleaser.yaml
#
metadata:
  # Whether to enable the size reporting or not.
  report_sizes: true

  # Set the modified timestamp on the metadata files, typically
  #
  # Templates: allowed.
  # Since: v1.20.
  mod_timestamp: "{{ .CommitTimestamp }}"
```

!!! info

    In versions 1.18 and 1.19, the `report_sizes` property is at the root of the
    yaml, instead of being under `metadata`.

## Report Sizes

You might want to enable this if you want to keep an eye on your binary/package
sizes.

It'll report the size of each artifact of the following types to the build
output, as well as on `dist/artifacts.json`:

- `Binary`
- `UniversalBinary`
- `UploadableArchive`
- `PublishableSnapcraft`
- `LinuxPackage`
- `CArchive`
- `CShared`
- `Header`
