# Mastodon

For it to work, you'll need to create a new Mastodon app (https://social.yourdomain.tld/settings/applications/new), and set
some environment variables on your pipeline:

- `MASTODON_SERVER`
- `MASTODON_CLIENT_ID`
- `MASTODON_CLIENT_SECRET`
- `MASTODON_ACCESS_TOKEN`

Then, you can add something like the following to your `.goreleaser.yaml` config:

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
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
