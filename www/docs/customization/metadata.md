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

  # Default git author used to commit to AUR, Homebrew, Winget, Nix, etc.
  #
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.12 -->.
  commit_author:
    # Git author name.
    #
    # Templates: allowed.
    name: goreleaserbot

    # Git author email.
    #
    # Templates: allowed.
    email: bot@goreleaser.com

    # Git commit signing configuration.
    # Only works if repository is
    signing:
      # Enable commit signing.
      enabled: true

      # The signing key to use.
      # Can be a key ID, fingerprint, email address, or path to a key file.
      #
      # Templates: allowed.
      key: "{{ .Env.GPG_SIGNING_KEY }}"

      # The GPG program to use for signing.
      #
      # Templates: allowed.
      program: gpg2

      # The signature format to use.
      #
      # Valid options: openpgp, x509, ssh.
      # Default: openpgp.
      format: openpgp

{% include-markdown "../includes/commit_author.md" comments=false start='---\n\n' %}
```

<!-- md:templates -->
