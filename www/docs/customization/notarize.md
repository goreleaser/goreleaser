# Notarize macOS binaries

GoReleaser can sign & notarize macOS binaries
(and [Universal Binaries][unibin]) using [anchore/quill][quill].

To use it, you'll need:

- An [Apple Developer Account](https://developer.apple.com/) ($99/year).
- A [certificate](https://developer.apple.com/account/resources/certificates/add)
  from said account. It should be of "Developer ID Application" type.
  This will give you a `.cer` file. You'll need to import it into KeyChain.app,
  and then export it as a `.p12` file. It'll will have a password.
- An App Store Connect
  [API key](https://appstoreconnect.apple.com/access/integrations/api/new).
  This will give you a `.p8` file.

So you should end up with:

1. a `Certificates.p12` file and the password to open it
1. a `ApiKey_AAABBBCCC.p8` file

Read the commented configuration excerpt below to learn how to use these files.

```yaml
# .goreleaser.yaml
notarize:
  macos:
    - # Whether this configuration is enabled or not.
      #
      # Default: false.
      # Templates: allowed.
      enabled: '{{ isEnvSet "MACOS_SIGN_P12" }}'

      # IDs to use to filter the built binaries.
      #
      # Default: the project name.
      ids:
        - build1
        - build2

      # Before notarizing, we need to sign the binary.
      # This blocks defines the configuration for doing so.
      sign:
        # The .p12 certificate file path or its base64'd contents.
        certificate: "{{.Env.MACOS_SIGN_P12}}"

        # The password to be used to open the certificate.
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"

      # Then, we notarize the binaries.
      notarize:
        # The issuer ID.
        # Its the UUID you see when creating the App Store Connect key.
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"

        # Key ID.
        # You can see it in the list of App Store Connect Keys.
        # It will also be in the ApiKey filename.
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"

        # The .p8 key file path or its base64'd contents.
        key: "{{.Env.MACOS_NOTARY_KEY}}"

        # Whether to wait for the notarization to finish.
        # Not recommended, as it could take a really long time.
        wait: true

        # Timeout for the notarization.
        # Beware of the overall `--timeout` time.
        # This only has any effect if `wait` is true.
        #
        # Default: 10m.
        timeout: 20m
```

{% include-markdown "../includes/templates.md" comments=false %}

!!! tip "base64"

    To base64 a file, you run this:

    ```bash
    base64 -w0 < ./Certificates.p12
    base64 -w0 < ./ApiKey_AAABBBCCC.p8
    ```

## Signing only

> Since v2.1.

If you want to only sign the binaries, but not notarize them, you can simply
leave the `notarize` section of your configuration empty.

[unibin]: ./universalbinaries.md
[quill]: https://github.com/anchore/quill
