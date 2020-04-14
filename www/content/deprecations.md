---
title: Deprecation notices
menu: true
weight: 500
hideFromIndex: true
---

This page will be used to list deprecation notices across GoReleaser.

Deprecate code will be removed after ~6 months from the time it was deprecated.

You can check your use of deprecated configurations by running:

```sh
$ goreleaser check
```

## Active deprecation notices

<!--

Template for new deprecations:

### property

> since yyyy-mm-dd

Description.

Change this:

```yaml
```

to this:

```yaml
```

-->

## Expired deprecation notices

The following options were deprecated for ~6 months and are now fully removed.

### puts

> since 2019-11-15, removed 2020-04-14 (v0.132.0)

The HTTP upload support was extended to also accept `POST` as a method,
so the name `puts` kind of lost its meaning.

Change this:

```yaml
puts:
- ...
```

to this:

```yaml
uploads:
- ...
```

Also note that secrets environment variable name prefixes have changed from
`PUT_` to `UPLOAD_`.

### nfpms.name_template

> since 2019-11-15, removed 2020-04-14 (v0.132.0)

The `name_template` field was deprecated in favor of a more clear one,
`file_name_template`.

Change this:

```yaml
nfpms:
  - name_template: foo
```

to this:

```yaml
nfpms:
  - file_name_template: foo
```

### blob

> since 2019-08-02, removed 2020-03-22 (v0.130.0)

Blob was deprecated in favor of its plural form.
It was already accepting multiple inputs, but its pluralized now so its more
clear.

Change this:

```yaml
blob:
  # etc
```

to this:

```yaml
blobs:
  # etc
```

### sign

> since 2019-07-20, removed 2020-03-22 (v0.130.0)

Sign was deprecated in favor of its plural form.

Change this:

```yaml
sign:
  # etc
```

to this:

```yaml
signs:
  -
    # etc
```

### brew

> since 2019-06-09, removed 2020-01-26 (v0.125.0)

Brew was deprecated in favor of its plural form.

Change this:

```yaml
brew:
  # etc
```

to this:

```yaml
brews:
  -
    # etc
```

### s3

> since 2019-06-09, removed 2020-01-07 (v0.125.0)

S3 was deprecated in favor of the new `blob`, which supports S3, Azure Blob and
GCS.

Change this:

```yaml
s3:
-
  # etc
```

to this:

```yaml
blobs:
-
  provider: s3
  # etc
```

ACLs should be set on the bucket, the `acl` option does not exist anymore.

### archive

> since 2019-04-16, removed 2019-12-27 (v0.124.0)

We now allow multiple archives, so the `archive` statement will be removed.

Change this:

```yaml
archive:
  format: zip
```

to this:

```yaml
archives:
  - id: foo
    format: zip
```

### snapcraft

> since 2019-05-27, removed 2019-12-27 (v0.124.0)

We now allow multiple Snapcraft configs, so the `snapcraft` statement will be removed.

Change this:

```yaml
snapcraft:
  publish: true
  # ...
```

to this:

```yaml
snapcrafts:
  -
    publish: true
    # ...
```

### nfpm

> since 2019-05-07, removed 2019-12-27 (v0.124.0)

We now allow multiple NFPM config, so the `nfpm` statement will be removed.

Change this:

```yaml
nfpm:
  formats:
    - deb
```

to this:

```yaml
nfpms:
  -
    formats:
      - deb
```

### docker.binary

> since 2018-10-01, removed 2019-08-02 (v0.114.0)

You can now create a Docker image with multiple binaries.

Change this:

```yaml
dockers:
- image: foo/bar
  binary: foo
```

to this:

```yaml
dockers:
- image: foo/bar
  binaries:
  - foo
```

### docker.image

> since 2018-10-20, removed 2019-08-02 (v0.114.0)

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

### docker.tag_templates

> since 2018-10-20, removed 2019-08-02 (v0.114.0)

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

### git.short_hash

> since 2018-10-03, removed 2019-01-19 (v0.98.0)

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

### fpm

> since 2018-02-17, removed 2017-08-15 (v0.83.0)

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

### docker.tag_template

> since 2018-01-19, removed 2017-08-15 (v0.83.0)

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

### docker.latest

> since 2018-01-19, removed 2017-08-15 (v0.83.0)

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
