# GemFury - apt and rpm repositories

<!-- md:pro -->

You can easily create `deb` and `yum` repositories on [Fury][fury] using GoReleaser.

## Usage

First, you need to create an account on [Fury][fury] and get a push token.

Then, you need to pass your account name to GoReleaser and have your push token
as an environment variable named `FURY_TOKEN`:

```yaml title=".goreleaser.yaml"
gemfury:
  - account: myaccount
```

This will automatically upload all your `deb` and `rpm` files.

## Customization

You can also have plenty of customization options:

```yaml title=".goreleaser.yaml"
gemfury:
  - # Fury account.
    # Config is skipped if empty
    account: "{{ .Env.FURY_ACCOUNT }}"

    # Skip this configuration in some conditions.
    #
    # Templates: allowed.
    disable: "{{ .IsNightly }}"

    # Environment variable name to get the push token from.
    # You might want to change it if you have multiple Fury configurations for
    # some reason.
    #
    # Default: 'FURY_TOKEN'.
    secret_name: MY_ACCOUNT_FURY_TOKEN

    # IDs to filter by.
    # configurations get uploaded.
    ids:
      - packages

    # Formats to upload.
    # Available options are `apk`, `deb`, and `rpm`.
    #
    # Default: ['apk', deb', 'rpm'].
    formats:
      - deb
      - apk # <!-- md:inline_version v2.7 -->.
```

[fury]: https://gemfury.com

<!-- md:templates -->
