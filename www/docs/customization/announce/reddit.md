# Reddit

For it to work, you'll need to [create a new Reddit app](https://www.reddit.com/prefs/apps), and set some environment
variables on your pipeline:

- `REDDIT_SECRET`
- `REDDIT_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  reddit:
    # Whether its enabled or not.
    enabled: true

    # Application ID for Reddit Application
    application_id: ""

    # Username for your Reddit account
    username: ""

    # URL template to use while publishing.
    #
    # Default: '{{ .ReleaseURL }}'.
    # Templates: allowed.
    url_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Title template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out!'.
    # Templates: allowed.
    title_template: ''GoReleaser {{ .Tag }} was just released!''
```

{% include-markdown "../../includes/templates.md" comments=false %}
