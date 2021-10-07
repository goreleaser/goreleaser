# Telegram

For it to work, you'll need to [create a new Telegram app](https://core.telegram.org/bots), and set
some environment variables on your pipeline:

- `TELEGRAM_TOKEN`

Also you need to know your channel's chat ID to talk with.

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  telegram:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Integer representation of your channel
    chat_id: 123456

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
