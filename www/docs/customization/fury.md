# Fury.io (apt and rpm repositories)

!!! success "GoReleaser Pro"

    The fury.io publisher is a [GoReleaser Pro feature](/pro/). You might be
    able to reproduce some of its behavior on GoReleaser OSS using
    [custom publishers](/customization/publishers/).

You can easily create `deb` and `yum` repositories on [fury.io][fury] using GoReleaser.

## Usage

First, you need to create an account on [fury.io][fury] and get a push token.

Then, you need to pass your account name to GoReleaser and have your push token
as an environment variable named `FURY_TOKEN`:

```yaml
# .goreleaser.yaml
furies:
  - account: myaccount
```

This will automatically upload all your `deb` and `rpm` files.

## Customization

You can also have plenty of customization options:

```yaml
# goreleaser.yaml

furies:
  - # fury.io account.
    # Config is skipped if empty
    account: "{{ .Env.FURY_ACCOUNT }}"

    # Skip the announcing feature in some conditions, for instance, when
    # publishing patch releases.
    # Any value different of 'true' will be considered 'false'.
    #
    # Templates: allowed
    skip: "{{gt .Patch 0}}"

    # Environment variable name to get the push token from.
    # You might want to change it if you have multiple fury configurations for
    # some reason.
    #
    # Default: 'FURY_TOKEN'
    secret_name: MY_ACCOUNT_FURY_TOKEN

    # IDs to filter by.
    # configurations get uploaded.
    ids:
      - packages

    # Formats to upload.
    # Available options are `deb` and `rpm`.
    #
    # Default: ['deb', 'rpm']
    formats:
      - deb
```

[fury]: https://gemfury.com

!!! tip

    Learn more about the [name template engine](/customization/templates/).
