# Slack

For it to work, you'll need to [create a new Incoming Webhook](https://api.slack.com/messaging/webhooks), and set some
environment variables on your pipeline:

- `SLACK_WEBHOOK`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  slack:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ trimsuffix .GitURL ".git" }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'

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
