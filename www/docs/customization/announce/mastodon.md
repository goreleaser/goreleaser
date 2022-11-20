# Mastodon

> Since: v1.13.0

For it to work, you'll need to create a new Mastodon app
`https://social.yourdomain.tld/settings/applications/new` with `write:statuses`
permissions, and set the following environment variables in your pipeline:

- `MASTODON_CLIENT_ID`: *"Client key"*.
- `MASTODON_CLIENT_SECRET`: *"Client secret"*.
- `MASTODON_ACCESS_TOKEN`: *"Your access token"*.

Then, you can add something like the following to your `.goreleaser.yaml`
configuration file:

```yaml
# .goreleaser.yaml
announce:
  mastodon:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
    message_template: 'Awesome project {{.Tag}} is out!'

    # Mastodon server URL.
    # Defaults to empty.
    server: https://mastodon.social
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
