# Discord

To use [Discord](https://discord.com/), you need
to [create a Webhook](https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks), and set following
environment variables on your pipeline:

- `DISCORD_WEBHOOK_ID`
- `DISCORD_WEBHOOK_TOKEN`

After this, you can add following section to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  discord:
    # Whether its enabled or not.
    enabled: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"

    # Set author of the embed.
    #
    # Default: 'GoReleaser'.
    author: ""

    # Color code of the embed. You have to use decimal numeral system, not hexadecimal.
    #
    # Default: '3888754' (the grey-ish from GoReleaser).
    color: ""

    # URL to an image to use as the icon for the embed.
    #
    # Default: 'https://goreleaser.com/static/avatar.png'.
    icon_url: ""
```

{% include-markdown "../../includes/templates.md" comments=false %}
