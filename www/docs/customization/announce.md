---
title: Announce
---

GoReleaser can also announce new releases to Twitter, Reddit, Slack, Discourse and Microsoft Teams.

It runs at the very end of the pipeline and can be skipped with the `--skip-announce` flag of
the [`release`](/cmd/goreleaser_release/) command.

## Twitter

For it to work, you'll need to [create a new Twitter app](https://developer.twitter.com/en/portal/apps/new), and set
some environment variables on your pipeline:

- `TWITTER_CONSUMER_KEY`
- `TWITTER_CONSUMER_SECRET`
- `TWITTER_ACCESS_TOKEN`
- `TWITTER_ACCESS_TOKEN_SECRET`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  twitter:
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'
```

## Teams

To use [Teams](https://www.microsoft.com/de-de/microsoft-teams/group-chat-software), you need
to [create a Webhook](https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook)
, and set following environment variable on your pipeline:

- `TEAMS_WEBHOOK`

After this, you can add following section to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  teams:
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # Title template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    title_template: 'GoReleaser {{ .Tag }} was just released!'

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
    message_template: 'Awesome project {{.Tag}} is out!'

    # Color code of the message. You have to use hexadecimal.
    # Defaults to `#2D313E` - the grey-ish from goreleaser
    color: ''

    # URL to an image to use as the icon for the message.
    # Defaults to `https://goreleaser.com/static/avatar.png`
    icon_url: ''
```

## Discord

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
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
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

## Slack

For it to work, you'll need to [create a new Incoming Webhook](https://api.slack.com/messaging/webhooks), and set some
environment variables on your pipeline:

- `SLACK_WEBHOOK`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  slack:
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .GitURL }}/releases/tag/{{ .Tag }}`
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

!!! tip Learn more about the [name template engine](/customization/templates/).

## Reddit

For it to work, you'll need to [create a new Reddit app](https://www.reddit.com/prefs/apps), and set some environment
variables on your pipeline:

- `REDDIT_SECRET`
- `REDDIT_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  reddit:
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # Application ID for Reddit Application
    application_id: ""

    # Username for your Reddit account
    username: ""

    # URL template to use while publishing.
    # Defaults to `{{ .GitURL }}/releases/tag/{{ .Tag }}`
    url_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Title template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    title_template: ''GoReleaser {{ .Tag }} was just released!''
```

## SMTP

For it to work, you'll need to set some environment variables on your pipeline:

- `SMTP_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  smtp:
    # Wether its enabled or not.
    # Defaults to false.
    enabled: true

    # SMTP Host
    host: "smtp.gmail.com"

    # SMTP Port
    port: 587

    # Sender of the email
    from: ""

    # Receivers of the email
    to:
      - ""
      - ""

    # Owner of the email
    username: ""

    # Body template to use within the email.
    # Defaults to `You can view details from: {{ .GitURL }}/releases/tag/{{ .Tag }}`
    body_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Subject template to use within the email subject.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    subject_template: ''GoReleaser {{ .Tag }} was just released!''
```
