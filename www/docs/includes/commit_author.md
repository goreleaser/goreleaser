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
