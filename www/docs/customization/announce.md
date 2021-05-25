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
