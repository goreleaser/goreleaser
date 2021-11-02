# Discord

To use [Discord](https://discord.com/), you need
to [create a Webhook](https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks), and set following
environment variables on your pipeline:

- `DISCORD_WEBHOOK_ID`
- `DISCORD_WEBHOOK_TOKEN`

After this, you can add following section to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  discord:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ trimsuffix .GitURL ".git" }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'

    # Set author of the embed.
    # Defaults to `GoReleaser`
    author: ''

    # Color code of the embed. You have to use decimal numeral system, not hexadecimal.
    # Defaults to `3888754` - the grey-ish from goreleaser
    color: ''

    # URL to an image to use as the icon for the embed.
    # Defaults to `https://goreleaser.com/static/avatar.png`
    icon_url: ''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
