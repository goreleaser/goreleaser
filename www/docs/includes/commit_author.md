---
search:
  exclude: true
---

    # Git author used to commit to the repository.
    #
    # Default: inferred from global metadata (Since v2.12).
    commit_author:
      # Git author name.
      #
      # Templates: allowed.
      name: goreleaserbot

      # Git author email.
      #
      # Templates: allowed.
      email: bot@goreleaser.com

      # Use GitHub App token for signed commits.
      # When enabled, the committer field is omitted from API calls,
      # allowing GitHub to automatically sign commits with the GitHub App identity.
      # See: https://docs.github.com/en/authentication/managing-commit-signature-verification/about-commit-signature-verification#signature-verification-for-bots
      #
      # <!-- md:inline_version v2.13 -->.
      # Default: false.
      use_github_app_token: false

      # Git commit signing configuration.
      # Only useful if repository is of type 'git'.
      #
      # <!-- md:inline_version v2.11 -->.
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
