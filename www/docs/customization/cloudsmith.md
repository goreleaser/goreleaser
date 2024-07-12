# Cloudsmith - apt, rpm, and alpine repositories

> Since v2.1 (Pro).

{% include-markdown "../includes/pro.md" comments=false %}

You can easily create `deb`, `alpine`, and `yum` repositories on
[Cloudsmith][cloudsmith] using GoReleaser.

## Usage

First, you need to create an account on [Cloudsmith][cloudsmith] and get an API
token.

Then, you need to pass your account name to GoReleaser and have your push token
as an environment variable named `CLOUDSMITH_TOKEN`:

```yaml
# .goreleaser.yaml
furies:
  - organization: myorg
    repository: myrepo
    distributions:
      deb: "ubuntu/xenial"
      alpine: "alpine/v3.8"
      rpm: "el/7"
```

This will automatically upload all your `apk`, `deb`, and `rpm` files.

## Customization

You can also have plenty of customization options:

```yaml
# goreleaser.yaml

cloudsmiths:
  - # Cloudsmith organization.
    # Config is skipped if empty
    #
    # Templates: allowed.
    organization: "{{ .Env.CLOUDSMITH_ORG }}"

    # Cloudsmith repository.
    # Config is skipped if empty
    #
    # Templates: allowed.
    organization: "{{ .ProjectName }}"

    # Skip the announcing feature in some conditions, for instance, when
    # publishing patch releases.
    # Any value different of 'true' will be considered 'false'.
    #
    # Templates: allowed.
    skip: "{{gt .Patch 0}}"

    # Environment variable name to get the push token from.
    # You might want to change it if you have multiple Cloudsmith configurations
    # for some reason.
    #
    # Default: 'CLOUDSMITH_TOKEN'.
    secret_name: MY_ACCOUNT_CLOUDSMITH_TOKEN

    # IDs to filter by.
    ids:
      - packages

    # Formats to upload.
    # Available options are `apk`, `deb`, and `rpm`.
    #
    # Default: ['apk', 'deb', 'rpm'].
    formats:
      - deb

    # Map of which distribution to use for each format.
    # Publish will be skipped if this is empty/not found.
    distributions:
      deb: "ubuntu/xenial"
      rpm: "el/7"
      alpine: "alpine/v3.8"
```

[cloudsmith]: https://cloudsmith.io/

{% include-markdown "../includes/templates.md" comments=false %}
