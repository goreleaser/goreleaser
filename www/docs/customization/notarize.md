# Notarize macOS binaries

GoReleaser can sign & notarize macOS binaries
(and [Universal Binaries][unibin]) using [anchore/quill][quill].

To use it, you'll need:

- An Apple Developer Account ($99/year).
- A [certificate](https://developer.apple.com/account/resources/certificates/add)
  from said account. It should be of "Developer ID Installer" type.
  This will give you a `.cer` file. You'll need to import it into KeyChain, then
  export it as a `.p12` file.
- An App Store Connect
  [key](https://appstoreconnect.apple.com/access/integrations/api/new).
  This should give you a `.p8` file.

```yaml
# .goreleaser.yaml
notarize:
  macos:
    - # Whether this configuration is enabled or not.
      #
      # Default: false
      # Templates: allowed
      enabled: '{{ isEnvSet "MACOS_SIGN_P12 }}'

      # IDs to use to filter the built binaries.
      #
      # Default: Project Name
      ids:
        - build1
        - build2

      # Before notarizing, we need to sign the binary.
      # This blocks defines the configuration for doing so.
      sign:
        # The .p12 certificate file path or base64'd contents.
        certificate: "{{.Env.MACOS_SIGN_P12}}"

        # The password used to open the certificate.
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"

      # Then, we notarize the binaries.
      notarize:
        # The issuer id.
        # The UUID you see when creating the App Store Connect key.
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"

        # Key ID.
        # You can see it in the list of App Store Connect Keys.
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"

        # The .p8 key file path or base64'd contents.
        key: "{{.Env.MACOS_NOTARY_KEY}}"

        # Whether to wait for the notarization to finish.
        # Not recommended, as it could take a really long time.
        wait: true

        # Timeout for the notarization, if wait is true.
        #
        # Default: 10m
        timeout: 20m
```

!!! tip

    Learn more about the [name template engine](/customization/templates/).

[unibin]: ./universalbinaries.md
[quill]: https://github.com/anchore/quill
