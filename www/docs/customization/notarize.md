# Notarize macOS applications

GoReleaser has two ways to do notarization for macOS:

1. Cross-platform using [anchore/quill][quill];
2. Native using `codesign` and `xcrun` (only on macOS);

The first can be used with binaries/[universal binaries][unibin] only.
Note that putting a signed and notarized binary inside a non-notarized `.app`
does not work!

The second is the recommended way if you need to ship
[App Bundles][appbundles] inside [DMGs][DMG].

## Getting the keys

To use these features, you'll need:

- An [Apple Developer Account](https://developer.apple.com/) ($99/year).
- A [certificate](https://developer.apple.com/account/resources/certificates/add)
  from said account. It should be of "Developer ID Application" type for DMGs,
  or "Developer ID Installer" for Pkgs.
  This will give you a `.cer` file. You'll need to import it into
  `KeyChain.app`, and then export it as a `.p12` file. It'll have a
  password.
- An App Store Connect
  [API key](https://appstoreconnect.apple.com/access/integrations/api/new).
  This will give you a `.p8` file.

So you should end up with:

1. a `Certificates.p12` file and the password to open it
1. a `ApiKey_AAABBBCCC.p8` file

If you plan to use them in GitHub Actions (or another CI), you'll need to
`base64` encode them as well.

??? tip "base64 encoding"

    To base64 encode a file, you run this:

    ```bash
    base64 -w0 < ./Certificates.p12
    base64 -w0 < ./ApiKey_AAABBBCCC.p8
    ```

## Cross-platform

If you only need to sign and notarize your binaries, this is probably the best
alternative.

It has no external dependencies, and works on any operating system.

???+ danger "Do not use with App Bundles"

    Do not use this method if you create [App Bundles][appbundles].
    App Bundles in which only the binary is signed/notarized are deemed damaged
    by macOS.
    In that case, use the [native signing](#native) and notarizing documented
    below.

Read the commented configuration excerpt below to learn how to use do it.

```yaml title=".goreleaser.yaml"
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
      # This block defines the configuration for doing so.
      sign:
        # The .p12 certificate file path or its base64'd contents.
        #
        # Templates: allowed.
        certificate: "{{.Env.MACOS_SIGN_P12}}"

        # The password to be used to open the certificate.
        #
        # Templates: allowed.
        password: "{{.Env.MACOS_SIGN_PASSWORD}}"

        # Allows to set the signature entitlements XML file.
        #
        # Templates: allowed.
        # <!-- md:inline_version v2.6 -->.
        entitlements: ./path/to/entitlements.xml

      # Then, we notarize the binaries.
      #
      # You can leave this section empty if you only want
      # to sign the binaries (<!-- md:inline_version v2.1 -->).
      notarize:
        # The issuer ID.
        # Its the UUID you see when creating the App Store Connect key.
        #
        # Templates: allowed.
        issuer_id: "{{.Env.MACOS_NOTARY_ISSUER_ID}}"

        # Key ID.
        # You can see it in the list of App Store Connect Keys.
        # It will also be in the ApiKey filename.
        #
        # Templates: allowed.
        key_id: "{{.Env.MACOS_NOTARY_KEY_ID}}"

        # The .p8 key file path or its base64'd contents.
        #
        # Templates: allowed.
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

<!-- md:templates -->

### GitHub Actions

In this case, signing and notarizing inside GitHub Actions is just a matter of
adding the environment variables to the `goreleaser-action` setup.

<details>
  <summary>release.yml</summary>

```yaml title=".github/workflows/release.yml"
name: goreleaser
# ...

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    env:
      # The base64 of the contents of your '.p12' key.
      MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}

      # The password to open the '.p12' key.
      MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}

      # The base64 of the contents of your '.p8' key.
      MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}

      # The ID of the '.p8' key.
      # You can find it in the filename, as well as the Apple Developer Portal
      # website.
      MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}

      # The issuer UUID.
      # You can find it in the Apple Developer Portal website.
      MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
    steps:
      # ...
      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser-pro
          version: "~> v2"
          args: release --clean
```

</details>

## Native

<!-- md:version v2.8 -->
<!-- md:pro -->

This method can sign and notarize [App Bundles][appbundles] and
[macOS Pkgs][macospkg], but it depends on `xcrun`, `codesign`, and
`productsign`.

It works with both [DMGs][DMG] and [macOS Pkgs][macospkg].

See the configuration options below.

```yaml title=".goreleaser.yaml"
notarize:
  macos_native:
    - # Whether this configuration is enabled or not.
      #
      # Default: false.
      # Templates: allowed.
      enabled: "true"

      # IDs to use to filter the built binaries.
      #
      # Default: the project name.
      ids:
        - build1
        - build2

      # Which artifact type this config applies to.
      # Valid options are "dmg" (default) and "pkg".
      #
      # When "dmg": signs AppBundle with codesign, notarizes DMG.
      # When "pkg": signs MacOSPkg with productsign, notarizes MacOSPkg.
      #
      # Default: "dmg".
      # <!-- md:inline_version v2.14-unreleased -->.
      use: dmg

      # Before notarizing, we need to sign the artifact.
      # This block defines the configuration for doing so.
      sign:
        # The path to the Keychain, if needed.
        #
        # Templates: allowed.
        keychain: "{{ .Env.KEYCHAIN_PATH }}"

        # The identity in Keychain.
        # For DMGs, use "Developer ID Application: Name".
        # For Pkgs, use "Developer ID Installer: Name".
        #
        # Templates: allowed.
        identity: "Developer ID Application: Carlos Becker"

        # Options to pass to 'codesign' (only used for DMGs).
        # You will generally want to add 'runtime' here.
        options: [runtime]

        # Allows to set the signature entitlements XML file (only used for DMGs).
        #
        # Templates: allowed.
        entitlements: ./path/to/entitlements.xml

      # Then, we notarize the artifacts.
      notarize:
        # Profile name.
        #
        # Templates: allowed.
        profile_name: "{{ .Env.MACOS_NOTARY_PROFILE_NAME }}"

        # Whether to wait for the notarization to finish.
        # Not recommended, as it could take a really long time.
        wait: true
```

<!-- md:templates -->

??? tip "Creating a profile"

    To use this, you'll need to create a profile with `notarytool`.
    You can do so in your machine with:

    ```bash
    xcrun notarytool store-credentials "$MACOS_NOTARY_PROFILE_NAME" \
      --key "$KEY_PATH" \
      --key-id "$MACOS_NOTARY_KEY_ID" \
      --issuer "$MACOS_NOTARY_ISSUER_ID" \
      --keychain $KEYCHAIN_PATH
    ```

### GitHub Actions

**This is only needed for native notarization.**

Make sure to read the [official GitHub Guide][gh-guide] as well, but this is how
we are doing it, in case you want to save some time.

You can also take a look at this
[live example](https://github.com/goreleaser/example-notarized-apps).

<details>
  <summary>release.yml</summary>

```yaml title=".github/workflows/release.yml"
name: goreleaser
# ...

jobs:
  goreleaser:
    runs-on: macos-latest # only on macos
    env:
      # The base64 of the contents of your '.p12' key.
      MACOS_SIGN_P12: ${{ secrets.MACOS_SIGN_P12 }}

      # The password to open the '.p12' key.
      MACOS_SIGN_PASSWORD: ${{ secrets.MACOS_SIGN_PASSWORD }}

      # A password for our temporary keychain
      KEYCHAIN_PASSWORD: ${{ secrets.KEYCHAIN_PASSWORD }}

      # The profile name to create and use for notarization.
      MACOS_NOTARY_PROFILE_NAME: ${{ secrets.MACOS_NOTARY_PROFILE_NAME }}

      # The base64 of the contents of your '.p8' key.
      MACOS_NOTARY_KEY: ${{ secrets.MACOS_NOTARY_KEY }}

      # The ID of the '.p8' key.
      # You can find it in the filename, as well as the Apple Developer Portal
      # website.
      MACOS_NOTARY_KEY_ID: ${{ secrets.MACOS_NOTARY_KEY_ID }}

      # The issuer UUID.
      # You can find it in the Apple Developer Portal website.
      MACOS_NOTARY_ISSUER_ID: ${{ secrets.MACOS_NOTARY_ISSUER_ID }}
    steps:
      # ...
      - name: "setup-keychain"
        run: |
          # create variables
          CERTIFICATE_PATH=$RUNNER_TEMP/goreleaser.p12
          KEY_PATH=$RUNNER_TEMP/goreleaser.p8
          KEYCHAIN_PATH=$RUNNER_TEMP/goreleaser.keychain-db

          # import certificate and key from secrets
          echo -n "$MACOS_SIGN_P12" | base64 --decode -o $CERTIFICATE_PATH
          echo -n "$MACOS_NOTARY_KEY" | base64 --decode -o $KEY_PATH

          # create temporary keychain
          security create-keychain -p "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH
          security set-keychain-settings -lut 21600 $KEYCHAIN_PATH
          security unlock-keychain -p "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH

          # import certificate to keychain
          security import $CERTIFICATE_PATH -P "$MACOS_SIGN_PASSWORD" -A -t cert -f pkcs12 -k $KEYCHAIN_PATH
          security set-key-partition-list -S apple-tool:,apple: -k "$KEYCHAIN_PASSWORD" $KEYCHAIN_PATH
          security list-keychain -d user -s $KEYCHAIN_PATH

          # create notary profile
          xcrun notarytool store-credentials "$MACOS_NOTARY_PROFILE_NAME" \
            --key "$KEY_PATH" \
            --key-id "$MACOS_NOTARY_KEY_ID" \
            --issuer "$MACOS_NOTARY_ISSUER_ID" \
            --keychain $KEYCHAIN_PATH

          # export the keychain path
          echo "KEYCHAIN_PATH=$KEYCHAIN_PATH" >>$GITHUB_ENV

      # ...

      - uses: goreleaser/goreleaser-action@v6
        with:
          distribution: goreleaser-pro
          version: "~> v2"
          args: release --clean
```

</details>

## How it works

To make the behavior of this featur a bit clearer, this is the order in which
the relevant steps are executed:

=== "Cross-platform"

    The cross-platform version uses [quill][] under the hood.
    It is imported as a Go library and built into GoReleaser, so this just
    works.

    ```mermaid
    graph LR
      A[Create Binaries] --> B[Sign Binaries]
      B --> C[Notarize Binaries]
    ```

    Once the binaries are built, the notary step does everything in a single
    run.
    The signed binaries are then used from that point forward.

=== "Native"

    The native version runs `codesign` and `xcrun notarytool`.
    It only works on macOS and needs access to a Keychain.

    ```mermaid
    graph LR
      A[Create Binaries] --> B[Create App Bundles]
      B --> C[Sign App Bundles]
      C --> D[Create DMGs]
      D --> E[Notarize DMGs]
    ```

    Here things get a little bit more complicated.
    First, it only signs App Bundles, so they need to be created first.
    Once the App Bundle is signed, it needs to be packaged in a DMG.
    Finally, the DMG is notarized and used from that point on.

[unibin]: ./universalbinaries.md
[appbundles]: ./app_bundles.md
[quill]: https://github.com/anchore/quill
[DMG]: ./dmg.md
[macospkg]: ./pkg.md
[gh-guide]: https://docs.github.com/en/actions/use-cases-and-examples/deploying/installing-an-apple-certificate-on-macos-runners-for-xcode-development
