# DMG

<!-- md:pro -->

GoReleaser can create DMG images for macOS using `mkisofs` or `hdiutil`.

The `dmg` section specifies how the images should be created:

```yaml title=".goreleaser.yaml"
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

    # Which kind of artifact to use.
    #
    # Valid options are:
    # - 'binary':    binary
    # - 'appbundle': app bundles
    #
    # Default: 'binary'
    # <!-- md:inline_pro -->.
    # <!-- md:inline_version v2.4 -->.
    use: appbundle

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # <!-- md:inline_pro -->.
    # <!-- md:inline_version v2.4 -->.
    # Templates: allowed.
    if: '{{ eq .Os "linux" }}'

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1.
    goamd64: v1

    # More files that will be available in the context in which the image
    # will be built.
    extra_files:
      - logo.ico
      - glob: ./docs/*.md
      - glob: ./single_file.txt
        # Templates: allowed.
        # Note that this only works if glob matches exactly 1 file.
        name_template: file.txt

    # Additional templated extra files to add to the DMG.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the image as it would with the
    # extra_files field above.
    #
    # <!-- md:inline_pro -->.
    # <!-- md:inline_version v2.4 -->.
    # Templates: allowed.
    templated_extra_files:
      - src: LICENSE.tpl
        dst: LICENSE.txt
        mode: 0644

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

<!-- md:templates -->
