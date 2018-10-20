---
title: Deprecation notices
menu: true
weight: 500
hideFromIndex: true
---

This page will be used to list deprecation notices accross GoReleaser.

Deprecate code will be removed after ~6 months from the time it was deprecated.

# Active deprecation notices

## docker.image

> since 2018-10-20

This property was deprecated in favor of more flexible `image_templates`.
The idea is to be able to define several images and tags using templates instead of just one image with tag templates.
This flexibility allows images to be pushed to multiple registries.

Change this:

```yaml
dockers:
- image: foo/bar
  tag_templates:
    - '{{ .Tag }}'
```

to this:

```yaml
dockers:
- image_templates:
    - 'foo/bar:{{ .Tag }}'
```

## docker.tag_templates

> since 2018-10-20

This property was deprecated in favor of more flexible `image_templates`.
The idea is to be able to define several images and tags using templates instead of just one image with tag templates.

Change this:

```yaml
dockers:
- image: foo/bar
  tag_templates:
    - '{{ .Tag }}'
```

to this:

```yaml
dockers:
- image_templates:
    - 'foo/bar:{{ .Tag }}'
```


<!--

Template for new deprecations:

## property

> since yyyy-mm-dd

Description.

Change this:

```yaml
```

to this:

```yaml
```

 -->

## git.short_hash

> since 2018-10-03

This property was being used to tell GoReleaser to use short git hashes
instead of the full ones. This has been removed in favor of specific
template variables (`.FullCommit` and `.ShortCommit`).

Change this:

```yaml
git:
  short_hash: true

fake:
  foo_template: 'blah {{ .Commit }}'
```

to this:

```yaml
fake:
  foo_template: 'blah {{ .ShortCommit }}'
```

# Expired deprecation notices

The following options were deprecated for ~6 months and are now unsupported.

## fpm

> since 2018-02-17
>
> removed 2017-08-15

FPM is deprecated in favor of nfpm, which is a simpler alternative written
in Go. The objective is to remove the ruby dependency thus simplify the
CI/CD pipelines.

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

## docker.tag_template

> since 2018-01-19
>
> removed 2017-08-15

This property was deprecated in favor of the pluralized `tag_templates`.
The idea is to be able to define several tags instead of just one.

Change this:

```yaml
dockers:
- image: foo/bar
  tag_template: '{{ .Tag }}'
```

to this:

```yaml
dockers:
- image: foo/bar
  tag_templates:
    - '{{ .Tag }}'
```

## docker.latest

> since 2018-01-19
>
> removed 2017-08-15

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
