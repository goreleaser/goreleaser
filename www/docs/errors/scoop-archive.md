# Scoop requires single a windows archive

The Scoop pipe requires a Windows build and archive.

Usually, if you see this error, one of these 2 things probably happened:

## 1. Using binary archive format

The archive should not be in `binary` format.

For instance, this won't work:

```yaml
archives:
  - format: binary
```

But this would:

```yaml
archives:
  - format: zip
```

## 2. Multiple archives for the same GOOS/GOARCH

If you build multiple binaries and ship them in multiple archives, for example,
one for the _client_ and another one for the _server_ of a given project, you
will need to have multiple `scoops` in your configuration as well.

Scoops only allow to install a single archive per manifest, so we need to do
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

Also notice the `goamd64` options, it must match the one from your build.
By default, only `GOAMD64` `v1` is built.

Please refer to the [documentation](../customization/scoop.md) for more details.
