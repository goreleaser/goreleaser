# Source Archive

You may add the current tag source archive to the release as well. This is
particularly useful if you want to sign it, for example.

```yaml
# .goreleaser.yaml
source:
  # Whether this pipe is enabled or not.
  enabled: true

  # Name template of the final archive.
  #
  # Default: '{{ .ProjectName }}-{{ .Version }}'.
  # Templates: allowed.
  name_template: "{{ .ProjectName }}"

  # Format of the archive.
  #
  # Valid formats are: tar, tgz, tar.gz, and zip.
  #
  # Default: 'tar.gz'.
  format: "tar"

  # Prefix.
  # String to prepend to each filename in the archive.
  #
  # Templates: allowed.
  prefix_template: "{{ .ProjectName }}-{{ .Version }}/"

  # This will make the destination paths be relative to the longest common
  # path prefix between all the files matched and the source glob.
  # Enabling this essentially mimic the behavior of nfpm's contents section.
  # It will be the default by June 2023.
  rlcp: true

  # Additional files/globs you want to add to the source archive.
  #
  # Templates: allowed.
  files:
    - LICENSE.txt
    - README_{{.Os}}.md
    - CHANGELOG.md
    - docs/*
    - design/*.png
    - templates/**/*
    # a more complete example, check the globbing deep dive below
    - src: "*.md"
      dst: docs

      # Strip parent directories when adding files to the archive.
      strip_parent: true

      # File info.
      # Not all fields are supported by all formats available formats.
      # Default: file info of the source file.
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
  # This feature is only available in GoReleaser Pro.
  # Templates: allowed.
  templated_files:
    # a more complete example, check the globbing deep dive below
    - src: "LICENSE.md.tpl"
      dst: LICENSE.md
      info:
        owner: root
        group: root
        mode: 0644
        mtime: 2008-01-02T15:04:05Z
```

{% include-markdown "../includes/templates.md" comments=false %}
