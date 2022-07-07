# Mattermost

For it to work, you'll need to [create a new Incoming Webhook](https://docs.mattermost.com/developer/webhooks-incoming.html) in your own Mattermost deployment, and set some
environment variables on your pipeline:

- `MATTERMOST_WEBHOOK`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  mattermost:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Title template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    title_template: 'GoReleaser {{ .Tag }} was just released!'

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
    message_template: 'Awesome project {{.Tag}} is out!'

    # Color code of the message. You have to use hexadecimal.
    # Defaults to `#2D313E` - the grey-ish from goreleaser
    color: ''

    # The name of the channel that the user selected as a destination for webhook messages.
    channel: '#channel'

    # Set your Webhook's user name.
    username: ''

    # Emoji to use as the icon for this message. Overrides icon_url.
    icon_emoji: ''

    # URL to an image to use as the icon for this message.
    icon_url: ''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
