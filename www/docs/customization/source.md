# Source Archive

You may add the current tag source archive to the release as well. This is particularly
useful if you want to sign it, for example.

```yaml
# .goreleaser.yml
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
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
