---
title: macOS Universal Binaries
---

GoReleaser can create _macOS Universal Binaries_ - also known as _Fat Binaries_.
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
  # You will want to change this if you have multiple builds!
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

!!! warning
    You'll want to change `name_template` for each `id` you add in universal binaries, otherwise they'll have the same name.

    Example:

    ```yaml
    universal_binaries:
    - id: foo
      name_template: bin1
    - id: bar
      name_template: bin2
    ```
