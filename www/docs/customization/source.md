# Source Archive

You may add the current tag source archive to the release as well. This is
particularly useful if you want to sign it, for example.

```yaml
# .goreleaser.yaml
source:
  # Whether this pipe is enabled or not.
  # Defaults to `false`
  enabled: true

  # Name template of the final archive.
  # Defaults to `{{ .ProjectName }}-{{ .Version }}`
  name_template: '{{ .ProjectName }}'

  # Format of the archive.
  # Any format git-archive supports, this supports too.
  # Defaults to `tar.gz`
  format: 'tar'

  # Prefix template.
  # String to prepend to each filename in the archive.
  # Defaults to empty
  prefix_template: '{{ .ProjectName }}-{{ .Version }}/'

  # This will make the destination paths be relative to the longest common
  # path prefix between all the files matched and the source glob.
  # Enabling this essentially mimic the behavior of nfpm's contents section.
  # It will be the default by June 2023.
  #
  # Default: false
  # Since: v1.14.
  rlcp: true

  # Additional files/template/globs you want to add to the source archive.
  #
  # Default: empty.
  # Since: v1.11.
  files:
    - LICENSE.txt
    - README_{{.Os}}.md
    - CHANGELOG.md
    - docs/*
    - design/*.png
    - templates/**/*
    # a more complete example, check the globbing deep dive below
    - src: '*.md'
      dst: docs

      # Strip parent folders when adding files to the archive.
      # Default: false
      strip_parent: true

      # File info.
      # Not all fields are supported by all formats available formats.
      # Defaults to the file info of the actual file if not provided.
      info:
        owner: root
        group: root
        mode: 0644
        # format is `time.RFC3339Nano`
        mtime: 2008-01-02T15:04:05Z


  # Additional templated files to add to the source archive.
  # Those files will have their contents pass through the template engine,
  # and its results will be added to the source archive.
  #
  # Default: empty
  # Since: v1.17 (pro)
  # This feature is only available in GoReleaser Pro.
  files:
    # a more complete example, check the globbing deep dive below
    - src: 'LICENSE.md.tpl'
      dst: LICENSE.md
      info:
        owner: root
        group: root
        mode: 0644
        mtime: 2008-01-02T15:04:05Z

```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
