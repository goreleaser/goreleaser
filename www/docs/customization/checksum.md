# Checksums

GoReleaser will always generate checksums for artifacts being uploaded, except
if explicitly disabled.

The `checksum` section allows the following customizations:

```yaml
# .goreleaser.yaml
checksum:
  # You can change the name of the checksums file.
  #
  # Default: "{{ .ProjectName }}_{{ .Version }}_checksums.txt", or
  #          "{{ .ArtifactName }}.{{ .Algorithm }}" if "split" is set.
  # Templates: allowed
  name_template: "{{ .ProjectName }}_checksums.txt"

  # Algorithm to be used.
  # Accepted options are sha256, sha512, sha1, crc32, md5, sha224 and sha384.
  #
  # Default: sha256.
  algorithm: sha256

  # Allows to create one checksum file for each file being checksummed, instead
  # of a single file with all the checksums.
  # Note that the checksums created by this method will contain only the
  # checksum itself, without the filename.
  #
  # Since: v1.25
  split: true

  # Disable the generation/upload of the checksum file.
  disable: true
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).
