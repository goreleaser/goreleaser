---
title: MacOS Universal Binaries
---

GoReleaser can create _MacOS Universal Binaries_ - also known as _Fat Binaries_.
Those binaries are in a special format that contains both `arm64` and `amd64` executables in a single file.

Here's how to use it:

```yaml
# .goreleaser.yml
universal_binaries:
-
  # ID of the source build
  #
  # Defaults to the project name.
  id: foo

  # Universal binary name template.
  #
  # Defaults to '{{ .ProjectName }}'
  name_template: '{{.ProjectName}}_{{.Version}}'

  # Whether to remove the previous single-arch binaries from the artifact list.
  # If left as false, your end release might have both several macOS archives: amd64, arm64 and all.
  #
  # Defaults to false.
  replace: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

The minimal configuration for most setups would look like this:

```yaml
# .goreleaser.yml
universal_binaries:
- replace: true
```

That config will join your default build macOS binaries into an Universal Binary,
removing the single-arch binaries from the artifact list.

From there, the `Arch` template variable for this file will be `all`.
You can use the Go template engine to remove it if you'd like.
