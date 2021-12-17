# Extra Files

GoReleaser supports including extra pre-existing files in the following (each also supporting individual configuration):

- [blobs](/customization/blob/)
- [checksum](/customization/checksum/)
- [publishers](/customization/publishers/)
- [release](/customization/release/)

```yaml
# .goreleaser.yml
extra_files:
  # Multiple extra file configurations can be added.
  # Defaults to empty.
  -
    # Glob pattern to match file(s). Supports templates.
    # The filename will be the last part of the path (base).
    glob: ./path/to/file.txt

    # Replacement filename. Supports templates.
    # Glob must match single file for name replacement.
    name_template: newname.txt
```

If another file with the same name exists across globs, the last one found will be used.

```yaml
# .goreleaser.yml
extra_files:
  - glob: ./glob/**/to/**/file/**/*
  - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
```
