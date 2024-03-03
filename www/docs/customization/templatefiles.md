# Template Files

> Since v1.16 (pro)

!!! success "GoReleaser Pro"

    Template Files is a [GoReleaser Pro feature](/pro/).

Template Files allow you to create custom files and/or scripts using
GoReleaser's internal state and template variables, for example, an installer
script.

All the templated files are uploaded to the release by default.

```yaml
# .goreleaser.yaml
template_files:
  - # ID of this particular file.
    #
    # Default: 'default'
    id: default

    # Source path of the template file.
    # Ignored if empty.
    #
    # Templates: allowed
    src: foo.tpl.sh

    # Destination path of the file.
    # Will be prefixed with the `dist` folder.
    # Ignored if empty.
    #
    # Templates: allowed
    dst: foo.sh

    # File mode.
    #
    # Default: 0655.
    mode: 0755
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).
