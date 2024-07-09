# DMG

{% include-markdown "../includes/pro.md" comments=false %}

GoReleaser can create DMG images for macOS using `mkisofs` or `hdiutil`.

The `dmg` section specifies how the images should be created:

```yaml
# .goreleaser.yaml
dmg:
  - # ID of the resulting image.
    #
    # Default: the project name.
    id: foo

    # Filename of the image (without the extension).
    #
    # Default: '{{.ProjectName}}_{{.Arch}}'.
    # Templates: allowed.
    name: "myproject-{{.Arch}}"

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1.
    goamd64: v1

    # More files that will be available in the context in which the image
    # will be built.
    extra_files:
      - logo.ico

    # Whether to remove the archives from the artifact list.
    # If left as false, your end release will have both the archives and the
    # dmg files.
    replace: true

    # Set the modified timestamp on the output image, typically
    # you would do this to ensure a build was reproducible. Pass an
    # empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"
```

## Limitations

1. Due to the way symbolic links are handled on Windows, the `/Applications`
   link inside the image might not work if the image was built on Windows.
1. If running outside macOS, make sure to have `mkisofs` installed.

{% include-markdown "../includes/templates.md" comments=false %}
