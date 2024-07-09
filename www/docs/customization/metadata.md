# Metadata

GoReleaser creates some metadata files in the `dist` directory before it
finishes running.

You can also set some global defaults that can be used by other features.

```yaml
# .goreleaser.yaml
metadata:
  # Set the modified timestamp on the metadata files.
  #
  # Templates: allowed.
  mod_timestamp: "{{ .CommitTimestamp }}"

  # The maintainers of this software.
  # Most features will only use the first maintainer defined here.
  #
  # This feature is only available in GoReleaser Pro.
  # Since: v2.1 (pro).
  # Templates: allowed.
  maintainers:
    - "Foo Bar <foo at bar dot com>"

  # SPDX identifier of your app's license.
  #
  # This feature is only available in GoReleaser Pro.
  # Since: v2.1 (pro).
  # Templates: allowed.
  license: "MIT"

  # Your homepage.
  #
  # This feature is only available in GoReleaser Pro.
  # Since: v2.1 (pro).
  # Templates: allowed.
  homepage: "https://example.com/"

  # Your app's description.
  # Sometimes also referred as "short description".
  #
  # This feature is only available in GoReleaser Pro.
  # Since: v2.1 (pro).
  # Templates: allowed.
  description: "Software to create fast and easy drum rolls."

  # Your app's full description, sometimes also referred to as "long description".
  #
  # It can be a string directly, or you can use `from_url` or `from_file` to
  # source it from somewhere else.
  #
  # This feature is only available in GoReleaser Pro.
  # Since: v2.1 (pro).
  # Templates: allowed.
  full_description:
    # Loads from an URL.
    from_url:
      # Templates: allowed.
      url: https://foo.bar/README.md
      headers:
        x-api-token: "${MYCOMPANY_TOKEN}"

    # Loads from a local file.
    # Overrides `from_url`.
    from_file:
      # Templates: allowed.
      path: ./README.md
```

{% include-markdown "../includes/templates.md" comments=false %}
