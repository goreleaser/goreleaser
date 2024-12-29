# Metadata

GoReleaser creates some metadata files in the `dist` directory before it
finishes running.

You can also set some global defaults that can be used by other features.

```yaml title=".goreleaser.yaml"
metadata:
  # Set the modified timestamp on the metadata files.
  #
  # Templates: allowed.
  mod_timestamp: "{{ .CommitTimestamp }}"

  # The maintainers of this software.
  # Most features will only use the first maintainer defined here.
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.1 -->.
  # Templates: allowed.
  maintainers:
    - "Foo Bar <foo at bar dot com>"

  # SPDX identifier of your app's license.
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.1 -->.
  # Templates: allowed.
  license: "MIT"

  # Your homepage.
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.1 -->.
  # Templates: allowed.
  homepage: "https://example.com/"

  # Your app's description.
  # Sometimes also referred as "short description".
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.1 -->.
  # Templates: allowed.
  description: "Software to create fast and easy drum rolls."

  # Your app's full description, sometimes also referred to as "long description".
  #
  # It can be a string directly, or you can use `from_url` or `from_file` to
  # source it from somewhere else.
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.1 -->.
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

<!-- md:templates -->
