---
title: Fury
---

With [GoReleaser Pro](/pro) you can easily create deb and yum repositories on [fury.io][fury].

## Usage

First, you need to create an account on [fury.io][fury] and get a push token.

Then, you need to pass your account name to GoReleaser and have your push token as an environment variable named `FURY_TOKEN`:

```yaml
# .goreleaser.yml
furies:
- account: myaccount
```

This will automatically upload all your `deb` and `rpm` files.

## Customization

You can also have plenty of customization options:

```yaml
# goreleaser.yml

furies:
  -
    # fury.io account.
    # Config is skipped if empty
    account: my-account

    # Encironment variable name to get the push token from.
    # You might want to change it if you have multiple fury configurations for some reason.
    # Defaults to `FURY_TOKEN`.
    secret_name: MY_ACCOUNT_FURY_TOKEN

    # IDs to filter by.
    # Defaults to empty, which means all packages created by all nfpm configurations get uploaded.
    ids: [ 'nfpm1' ]

    # Formats to upload.
    # Available options are `deb` and `rpm`.
    # Defaults to `deb` and `rpm`.
    formats: [ 'deb' ]
```

!!! info
    The fury publisher is a [GoReleaser Pro feature](/pro/).
    You might be able to reproduce some of its behavior on GoReleaser OSS using [custom publishers](/customization/publishers/).

[fury]: https://fury.io/
