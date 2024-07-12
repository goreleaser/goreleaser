# Telegram

For it to work, you'll need to
[create a new Telegram app](https://core.telegram.org/bots), and set
some environment variables on your pipeline:

- `TELEGRAM_TOKEN`

Also you need to know your channel's chat ID to talk with.

Then, you can add something like the following to your `.goreleaser.yaml`
config:

```yaml
# .goreleaser.yaml
announce:
  telegram:
    # Whether its enabled or not.
    enabled: true

    # Integer representation of your channel
    #
    # Templates: allowed.
    chat_id: 123456

    # Message template to use while publishing.
    #
    # Default: '{{ mdv2escape .ProjectName }} {{ mdv2escape .Tag }} is out{{ mdv2escape "!" }} Check it out at {{ mdv2escape .ReleaseURL }}'.
    # Templates: allowed.
    message_template: 'Awesome project {{.Tag}} is out{{ mdv2escape "!" }}'

    # Parse mode.
    #
    # Valid options are MarkdownV2 and HTML.
    #
    # Default: 'MarkdownV2'.
    parse_mode: HTML
```

You can format your message using `MarkdownV2`, for reference, see the
[Telegram Bot API](https://core.telegram.org/bots/api#markdownv2-style).

{% include-markdown "../../includes/templates.md" comments=false %}
