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

    # More files that will be available in the context in which the image
    # will be built.
    extra_files:
      - README.md

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
```

## Limitations

1. As of v2.4, App Bundles can only be used together with [DMGs](dmg.md). This
   might change in the future.

<!-- md:templates -->
