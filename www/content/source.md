---
title: Source Archive
series: customization
hideFromIndex: true
weight: 41
---

You may add the current tag source archive to the release as well. This is particularly
useful if you want to sign it, for example.

```yml
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
```

> Learn more about the [name template engine](/templates).
