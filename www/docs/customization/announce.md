---
title: Announce
---

GoReleaser can also announce new releases, currently, to Twitter only.

It runs at the very end of the pipeline.

## Twitter

For it to work, you'll need to [create a new Twitter app](https://developer.twitter.com/en/portal/apps/new), and set some environment variables on your pipeline:

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

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## Reddit

For it to work, you'll need to [create a new Reddit app](https://www.reddit.com/prefs/apps), and set some environment variables on your pipeline:

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
