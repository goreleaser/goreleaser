---
title: "Telegram"
weight: 120
---

For it to work, you'll need to
[create a new Telegram bot](https://core.telegram.org/bots).

You should get a token, and export it as `TELEGRAM_TOKEN`.

You will also need to create a channel, and either its numerical ID or its
`@channelname`.

You'll need to add your bot as a channel admin, and give it "Post Messages"
permissions (it's inside the "Manage Messages" permission menu).

Then, you can add something like the following to your `.goreleaser.yaml`
configuration file:

```yaml {filename=".goreleaser.yaml"}
announce:
  telegram:
    # Whether it's enabled or not.
    #
    # Templates: allowed. {{< inline_version "v2.6" >}}
    enabled: true

    # Integer or `@` representation of your channel.
    #
    # Templates: allowed.
    chat_id: "@goreleasernews"

    # Specific thread ID to reply to.
    #
    # {{< inline_version "v2.15" >}}
    message_thread_id: 1234

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

> [!NOTE]
> If you use `MarkdownV2`, it's probably easier to do
> `{{ print "your message bits" | mdv2escape }}` to prevent issues.

You can also follow [our channel on Telegram](https://t.me/goreleasernews) if
you'd like.

{{< templates >}}
