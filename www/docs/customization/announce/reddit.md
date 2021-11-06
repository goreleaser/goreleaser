# Reddit

For it to work, you'll need to [create a new Reddit app](https://www.reddit.com/prefs/apps), and set some environment
variables on your pipeline:

- `REDDIT_SECRET`
- `REDDIT_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  reddit:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Application ID for Reddit Application
    application_id: ""

    # Username for your Reddit account
    username: ""

    # URL template to use while publishing.
    # Defaults to `{{ .ReleaseURL }}`
    url_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Title template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    title_template: ''GoReleaser {{ .Tag }} was just released!''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
