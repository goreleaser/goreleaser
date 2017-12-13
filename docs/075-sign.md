---
title: Sign
---

Goreleaser can sign the checksum files or all artifacts with [GPG](https://www.gnupg.org/)
so that your users can verify that the release was published by you.

Signing and checksumming work in tandem to ensure integrity (checksum) and
authenticity (signature) of your release.

```yml
# .goreleaser.yml
sign:
  # Provide the 16 byte id of your GPG signing key. 
  # This is not the same as the fingerprint.
  #
  # To get the id of the signing key run
  # 
  #   gpg --keyid-format LONG --list-keys
  #
  # pub   rsa2048/021E03CADDA53977 2013-07-06 [SCEA] [expires: 2019-11-07]
  #               ^^^^^^^^^^^^^^^^
  gpg_key_id = <16 byte id of the gpg signing key>

  # Path to the GPG keyring file.
  # gpg_keyring = '~/.gnupg/secring.gpg'

  # By default only checksum files are signed. To sign all artifacts
  # set this value to `true`.
  # sign_all_artifacts: false

  # Extension of the signature file.
  # signature_ext = '.asc' 
```
