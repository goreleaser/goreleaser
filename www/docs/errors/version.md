# Unsupported configuration version

This can show as an error or as an warning:

```
only configurations files on version: 2 are supported, yours is version: 0, please update your configuration
```

It has to do with the v2 update.

If you get it as a warning, your configuration file is valid in v2, but would
benefit with the version header.
You can remove the warning by adding this line to your configuration:

```yaml
version: 2
```

If you get it as a fatal error, it means your configuration is invalid.
You can still add the `version` header mentioned above, and it'll tell you which
parts of the configuration need to be fixed.

You can check the [deprecations](../deprecations.md) page to see how to fix
them.

Also worth reading the
[v2 announcement](../blog/posts/2024-06-04-goreleaser-v2.md), which contains an
upgrade guide.
