# macOS Universal Binaries

GoReleaser can create _macOS Universal Binaries_ - also known as _Fat Binaries_.
Those binaries are in a special format that contains both `arm64` and `amd64`
executables in a single file.

Here's how to use it:

```yaml
# .goreleaser.yaml
universal_binaries:
  - # ID of the resulting universal binary.
    #
    # Default: the project name.
    id: foo

    # IDs to use to filter the built binaries.
    #
    # Notice that you shouldn't include different apps' IDs here.
    # This field is usually only required if you are using CGO.
    #
    # Default: the value of the id field.
    ids:
      - build1
      - build2

    # Universal binary name.
    #
    # You will want to change this if you have multiple builds!
    #
    # Default: '{{ .ProjectName }}'.
    # Templates: allowed.
    name_template: "{{.ProjectName}}_{{.Version}}"

    # Whether to remove the previous single-arch binaries from the artifact list.
    # If left as false, your end release might have as much as three
    # archives for macOS: 'amd64', 'arm64' and 'all'.
    replace: true

    # Set the modified timestamp on the output binary, typically
    # you would do this to ensure a build was reproducible.
    # Pass an empty string to skip modifying the output.
    #
    # Templates: allowed.
    mod_timestamp: "{{ .CommitTimestamp }}"

    # Hooks can be used to customize the final binary,
    # for example, to run generators.
    #
    # Templates: allowed.
    hooks:
      pre: rice embed-go
      post: ./script.sh {{ .Path }}
```

{% include-markdown "../includes/templates.md" comments=false %}

For more info about hooks, see the [build section](./builds.md#build-hooks).

The minimal configuration for most setups would look like this:

```yaml
# .goreleaser.yml
universal_binaries:
  - replace: true
```

That config will join your default build macOS binaries into a Universal Binary,
removing the single-arch binaries from the artifact list.

From there, the `Arch` template variable for this file will be `all`.
You can use the Go template engine to remove it if you'd like.

!!! warning

    You'll want to change `name_template` for each `id` you add in universal
    binaries, otherwise they'll have the same name.

    Example:

    ```yaml
    universal_binaries:
    - id: foo
      name_template: bin1
    - id: bar
      name_template: bin2
    ```

## Naming templates

Most fields that support [templates](templates.md) will also
support the following build details:

<!-- to format the tables, use: https://tabletomarkdown.com/format-markdown-table/ -->

| Key     | Description                       |
| ------- | --------------------------------- |
| .Os     | `GOOS`, always `darwin`           |
| .Arch   | `GOARCH`, always `all`            |
| .Arm    | `GOARM`, always empty             |
| .Ext    | Extension, always empty           |
| .Target | Build target, always `darwin_all` |
| .Path   | The binary path                   |
| .Name   | The binary name                   |

!!! tip

    Notice that `.Path` and `.Name` will only be available after they are
    evaluated, so they are mostly only useful in the `post` hooks.
