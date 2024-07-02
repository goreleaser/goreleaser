# Checksums

GoReleaser generates a `project_1.0.0_checksums.txt` file and uploads it with the
release, so your users can validate if the downloaded files are correct.

The `checksum` section allows customizations of the filename:

```yaml
# .goreleaser.yaml
checksum:
  # You can change the name of the checksums file.
  #
  # Default: '{{ .ProjectName }}_{{ .Version }}_checksums.txt', or,
  #   when split is set: '{{ .ArtifactName }}.{{ .Algorithm }}'.
  # Templates: allowed.
  name_template: "{{ .ProjectName }}_checksums.txt"

  # Algorithm to be used.
  #
  # Accepted options are:
  # - sha256
  # - sha512
  # - sha1
  # - crc32
  # - md5
  # - sha224
  # - sha384
  # - sha3-256
  # - sha3-512
  # - sha3-224
  # - sha3-384
  # - blake2s
  # - blake2b
  #
  # Default: 'sha256'.
  algorithm: sha256

  # If true, will create one checksum file for each artifact.
  split: true

  # IDs of artifacts to include in the checksums file.
  #
  # If left empty, all published binaries, archives, linux packages and source archives
  # are included in the checksums file.
  ids:
    - foo
    - bar

  # Disable the generation/upload of the checksum file.
  disable: true

  # You can add extra pre-existing files to the checksums file.
  # The filename on the checksum will be the last part of the path (base).
  # If another file with the same name exists, the last one found will be used.
  #
  # Templates: allowed.
  extra_files:
    - glob: ./path/to/file.txt
    - glob: ./glob/**/to/**/file/**/*
    - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
    - glob: ./single_file.txt
      name_template: file.txt # note that this only works if glob matches 1 file only

  # Additional templated extra files to add to the checksum.
  # Those files will have their contents pass through the template engine,
  # and its results will be added to the checksum.
  #
  # This feature is only available in GoReleaser Pro.
  # Templates: allowed.
  templated_extra_files:
    - src: LICENSE.tpl
      dst: LICENSE.txt
```

{% include-markdown "../includes/templates.md" comments=false %}
