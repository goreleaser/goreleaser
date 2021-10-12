---
title: MacOS Fat Binaries
---

GoReleaser can create MacOS Fat Binaries - otherwise known as "Universal Binaries".
Those binaries contain both arm64 and amd64 binaries in a single binary.

Here's how to use it:

```yaml
# .goreleaser.yml
macos_fat_binaries:
-
  # ID of the source build
  # Defaults to the project name.
  id: foo

  # Fat binary name template.
  # Defaults to '{{ .ProjectName }}'
  name_template: '{{.ProjectName}}_{{.Version}}'

  # Wrther or not to remove the previous "thin" binaries from the artifact list.
  # Defaults to false.
  replace_plain_binaries: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

The minimal configuration for most setups would look like this:

```yaml
# .goreleaser.yml
macos_fat_binaries:
- replace_plain_binaries: true
```

That config will join your default build macOS binaries into a "fat" binary,
removing the "thin" binaries from the artifact list.

From there, the `Arch` template variable for this file will be `all`.
You can use the Go template engine to remove it if you'd like.
