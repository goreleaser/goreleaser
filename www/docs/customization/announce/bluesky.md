# Bluesky

To use [BlueSky](https://bsky.app/), you need
to create an account, and set the following
environment variable on your pipeline:

- `BLUESKY_ACCOUNT_PASSWORD`

After this, you can add following section to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  bluesky:
    # Whether it's enabled or not.
    enabled: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'
    # Templates: allowed
    message_template: 'Awesome project {{.Tag}} is out!'

    # The username of the account that will post
    # to BlueSky
    username: "my-project.bsky.social"

    # The Personal Data Server (PDS) to post to. If you don't know what that is, don't set it,
    # and it will default to the main BlueSky PDS
    pds_url: "https://my-custom-pds.example.com"

    # If using a custom PDS or if you have to go through a proxy, you may need to provide
    # custom CA certificates (recommended) or skip TLS verification (not recommended)
    ca_certs: |
      -----BEGIN CERTIFICATE-----
      MIi....
      -----END CERTIFICATE-----

    skip_tls_verify: false
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
