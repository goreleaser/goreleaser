# Telegram

For it to work, you'll need to
[create a new Telegram bot](https://core.telegram.org/bots).

You should get a token, which you should set export as `TELEGRAM_TOKEN`.

You will also need to create a channel, and either its numerical ID or its
`@channelname`.

Then, you can add something like the following to your `.goreleaser.yaml`
configuration file:

```yaml title=".goreleaser.yaml"
announce:
  telegram:
    # Whether its enabled or not.
    #
    # Templates: allowed (since v2.6).
    enabled: true

    # Integer or `@` representation of your channel.
    #
    # Templates: allowed.
    chat_id: "@goreleasernews"

    # Message template to use while publishing.
    #
    # Default: '{{ print .ProjectName " " .Tag " is out! Check it out at " .ReleaseURL | mdv2escape }}'.
    # Templates: allowed.
    message_template: '<a href="{{.ReleaseURL}}">{{ .ProjectName }} {{.Tag}}</a> is out!'

    # Parse mode.
    #
    # Valid options are 'MarkdownV2' and 'HTML'.
    #
    # Default: 'MarkdownV2'.
    parse_mode: HTML
```

You can format your message using `MarkdownV2` or `HTML`, for reference, see the
[Telegram Formatting Options documentation](https://core.telegram.org/bots/api#formatting-options).

!!! tip

    If you use `MarkdownV2`, its probably easier to do
    `{{ print "your message bits" | mdv2escape }}` to prevent issues.

<!-- md:templates -->
