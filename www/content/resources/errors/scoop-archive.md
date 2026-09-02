---
title: "Scoop requires a single Windows archive"
weight: 110
---

The Scoop pipe requires a Windows build and archive.

Usually, if you see this error, one of these two things probably happened:

## 1. Using binary archive format

The archive should not be in `binary` format.

For instance, this won't work:

```yaml
archives:
  - formats: [binary]
```

But this would:

```yaml
archives:
  - formats: [zip]
```

## 2. Multiple archives for the same GOOS/GOARCH

If you build multiple binaries and ship them in multiple archives, for example,
one for the _client_ and another one for the _server_ of a given project, you
will need to have multiple `scoops` in your configuration as well.

Scoop only allows installing a single archive per manifest, so we need to do
something like this:

```yaml
scoops:
  - ids: [client]
    name: foo
    # ...
  - ids: [server]
    name: food
    # ...
```

## Footnotes

Also notice the `goamd64` option: it must match the one from your build.
By default, only `GOAMD64` `v1` is built.

Please refer to the [documentation](/customization/publish/scoop/) for more details.
