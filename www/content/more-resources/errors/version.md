---
title: "Version-related configuration errors"
weight: 10
---

## Unsupported configuration version

This can show as an error or as an warning:

```
only version: 2 configuration files are supported, yours is version: 0, please update your configuration
```

It has to do with the v2 update.

If you get it as a warning, your configuration file is valid in v2, but would
benefit with the version header.
You can remove the warning by adding this line to your configuration:

```yaml {filename=".goreleaser.yml"}
version: 2
```

If you get it as a fatal error, it means your configuration is invalid.
You can still add the `version` header mentioned above, and it'll tell you which
parts of the configuration need to be fixed.

You can check the [deprecations](/more-resources/deprecations/) page to see how to fix
them.

Also worth reading the
[v2 announcement](/blog/posts/2024-06-04-goreleaser-v2/), which contains an
upgrade guide.

## Using a Pro configuration file with GoReleaser OSS

If you work with many people, you might not want to share the GoReleaser Pro key
with everyone.

Odds are, most of the people won't be doing much with it, maybe just testing the
builds and things like that.

If you want to allow GoReleaser OSS to read and use a GoReleaser Pro
configuration, set `pro: true` in your configuration file:

```yaml {filename=".goreleaser.yml"}
version: 2
pro: true
```

Then, whenever `--snapshot` is set, GoReleaser will happily proceed without
rejecting the configuration (it will warn about it, though).

> [!WARNING]
> When doing this, other YAML parsing errors might be ignored, such as fields
> that don't actually exist.
