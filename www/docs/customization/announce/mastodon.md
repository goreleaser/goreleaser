# Mastodon

For it to work, you'll need to create a new Mastodon app
`https://social.yourdomain.tld/settings/applications/new` with `write:statuses`
permissions, and set the following environment variables in your pipeline:

- `MASTODON_CLIENT_ID`: _"Client key"_.
- `MASTODON_CLIENT_SECRET`: _"Client secret"_.
- `MASTODON_ACCESS_TOKEN`: _"Your access token"_.

Then, you can add something like the following to your `.goreleaser.yaml`
configuration file:

```yaml
# .goreleaser.yaml
announce:
  mastodon:
    # Whether its enabled or not.
    enabled: true

    # Message to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"

    # Mastodon server URL.
    server: https://mastodon.social
```

{% include-markdown "../../includes/templates.md" comments=false %}
