# App Bundles

<!-- md:pro -->
<!-- md:version v2.4 -->

GoReleaser can create macOS App Bundles (a.k.a. `.app` files).

The `app_bundles` section specifies how the images should be created:

```yaml title=".goreleaser.yaml"
app_bundles:
  - # ID of the resulting image.
    #
    # Default: the project name.
    id: foo

    # Filename of the image (without the extension).
    #
    # Default: '{{.ProjectName}}'.
    # Templates: allowed.
    name: "myproject"

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
      - foo
      - bar

    # Allows to further filter the artifacts.
    #
    # Artifacts that do not match this expression will be ignored.
    #
    # Templates: allowed.
    if: '{{ eq .Os "linux" }}'

    # Icon file to use in the app.
    # Must be a `icns` file.
    #
    # Templates: allowed.
    icon: ./static/myapp.icns

    # App bundle name.
    #
    # Templates: allowed.
    bundle: com.example.myapp

    # Set the modified timestamp on the output image, typically
    # you would do this to ensure a build was reproducible. Pass an
    # empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"

    # Additional files/globs you want to add to the bundle.
    # You can use this to override the default generated 'Contents/Info.plist'
    # and/or to add more files.
    #
    # <!-- md:inline_version v2.6 -->.
    # Templates: allowed.
    extra_files:
      - src: ./release/Info.plist
        dst: Contents/Info.plist
      - src: ./release/icon.png
        dst: Contents/Resources/icon.png
        # File info.
        # Not all fields are supported by all formats available formats.
        #
        # Default: copied from the source file.
        info:
          # Must be in time.RFC3339Nano format.
          #
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"

    # Additional templated files to add to the app bundle.
    # Those files will have their contents pass through the template engine,
    # and its results will be added to the archive.
    # and/or to add more files.
    #
    # <!-- md:inline_version v2.6 -->.
    # Templates: allowed.
    # Extra template fields: `AppName`, `BinaryName`, and `Bundle`.
    templated_extra_files:
      # src can also be a glob, as long as dst is a directory.
      - src: "LICENSE.md.tpl"
        dst: LICENSE.md

        # File info.
        # Not all fields are supported by all formats available formats.
        #
        # Default: copied from the source file.
        info:
          # Must be in time.RFC3339Nano format.
          #
          # Templates: allowed.
          mtime: "{{ .CommitDate }}"
```

## Limitations

1. As of v2.4, App Bundles can only be used together with [DMGs](dmg.md). This
   might change in the future.
1. As of v2.6, even though the configuration allows `mode`, `owner`, and `group`
   in `extra_files` and `templated_extra_files`, those are not used. You should
   get a warning if you do so.

<!-- md:templates -->
