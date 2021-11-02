# Teams

To use [Teams](https://www.microsoft.com/de-de/microsoft-teams/group-chat-software), you need
to [create a Webhook](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook)
, and set following environment variable on your pipeline:

- `TEAMS_WEBHOOK`

After this, you can add following section to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  teams:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Title template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    title_template: 'GoReleaser {{ .Tag }} was just released!'

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ trimsuffix .GitURL ".git" }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'

    # Color code of the message. You have to use hexadecimal.
    # Defaults to `#2D313E` - the grey-ish from goreleaser
    color: ''

    # URL to an image to use as the icon for the message.
    # Defaults to `https://goreleaser.com/static/avatar.png`
    icon_url: ''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
