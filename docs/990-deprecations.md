---
title: Deprecation notices
---

This page will be used to list deprecation notices accross GoReleaser.

## docker.name_template

This property was deprecated in favor of the pluralized `name_templates`.

Change this:

```yaml
dockers:
- image: foo/bar
  name_template: '{{ .Tag }}'
```

to this:

```yaml
dockers:
- image: foo/bar
  tag_templates:
    - '{{ .Tag }}'
```

## docker.latest

The `latest` field in Docker config is deprecated in favor of the newer
`tag_templates` field.

Change this:

```yaml
dockers:
- image: foo/bar
  latest: true
```

to this:

```yaml
dockers:
- image: foo/bar
  tag_templates:
    - '{{ .Tag }}'
    - latest
```

## fpm

FPM is deprecated in favor of nfpm.

Just replace the `fpm` keyword by `nfpm` in your `goreleaser.yaml` file.

Change this:

```yaml
fpm:
  # ...
```

to this:

```yaml
nfpm:
  # ...
```
