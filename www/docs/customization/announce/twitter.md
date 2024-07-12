# Twitter

!!! warning
Twitter has [announced][tw] that API usage will no longer be free starting
Feb 9, 2023.

[tw]: https://twitter.com/TwitterDev/status/1621026986784337922

For it to work, you'll need to [create a new Twitter app](https://developer.twitter.com/en/portal/apps/new), and set
some environment variables on your pipeline:

- `TWITTER_CONSUMER_KEY`
- `TWITTER_CONSUMER_SECRET`
- `TWITTER_ACCESS_TOKEN`
- `TWITTER_ACCESS_TOKEN_SECRET`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  twitter:
    # Whether its enabled or not.
    enabled: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"
```

{% include-markdown "../../includes/templates.md" comments=false %}
