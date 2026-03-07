# Telegram

For it to work, you'll need to
[create a new Telegram app](https://core.telegram.org/bots), and set
some environment variables on your pipeline:

- `TELEGRAM_TOKEN`

Also you need to know your channel's chat ID to talk with.

Then, you can add something like the following to your `.goreleaser.yaml`
config:

```yaml title=".goreleaser.yaml"
announce:
  telegram:
    # Whether its enabled or not.
    #
    # Templates: allowed (since v2.6).
    enabled: true

    # Integer representation of your channel
    #
    # Templates: allowed.
    chat_id: 123456

    # Message template to use while publishing.
    #
    # Default: '{{ print .ProjectName " " .Tag " is out! Check it out at " .ReleaseURL | mdv2escape }}'.
    # Templates: allowed.
    message_template: 'Awesome project {{.Tag}} is out{{ mdv2escape "!" }}'

    # Parse mode.
    #
    # Valid options are MarkdownV2 and HTML.
    #
    # Default: 'MarkdownV2'.
    parse_mode: HTML
```

You can format your message using `MarkdownV2` or `HTML`, for reference, see the
[Telegram Formatting Options documentation](https://core.telegram.org/bots/api#formatting-options).

!!! tip

    If you use `MarkdownV2`, it's probably easier to do
    `{{ print "your message bits" | mdv2escape }}` to prevent issues.

<!-- md:templates -->
