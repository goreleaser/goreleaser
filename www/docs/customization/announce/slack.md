# Slack

For it to work, you'll need to [create a new Incoming Webhook](https://api.slack.com/messaging/webhooks), and set some
environment variables on your pipeline:

- `SLACK_WEBHOOK`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  slack:
    # Whether its enabled or not.
    enabled: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"

    # The name of the channel that the user selected as a destination for webhook messages.
    channel: "#channel"

    # Set your Webhook's user name.
    username: ""

    # Emoji to use as the icon for this message. Overrides icon_url.
    icon_emoji: ""

    # URL to an image to use as the icon for this message.
    icon_url: ""

    # Blocks for advanced formatting, see: https://api.slack.com/messaging/webhooks#advanced_message_formatting
    # and https://api.slack.com/messaging/composing/layouts#adding-blocks.
    #
    # Attention: goreleaser doesn't check the full structure of the Slack API: please make sure that
    # your configuration for advanced message formatting abides by this API.
    #
    # Templates: allowed.
    blocks: []

    # Attachments, see: https://api.slack.com/reference/messaging/attachments
    #
    # Attention: goreleaser doesn't check the full structure of the Slack API: please make sure that
    # your configuration for advanced message formatting abides by this API.
    #
    # Templates: allowed.
    attachments: []
```

{% include-markdown "../../includes/templates.md" comments=false %}
