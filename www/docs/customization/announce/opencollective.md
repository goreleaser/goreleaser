# OpenCollective

For it to work, you'll need to create a personal token (`https://opencollective.com/<user>/admin/for-developers`) and set the environment variable on your pipeline:

- `OPENCOLLECTIVE_TOKEN`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  opencollective:
    # Whether its enabled or not.
    enabled: true

    # Collective slug
    # https://opencollective.com/<slug>
    slug: "goreleaser"

    # Title for the update
    #
    # Default: '{{ .Tag }}'.
    # Templates: allowed.
    title_template: "Release of {{ .Tag }}"

    # Message to use while publishing. It can be HTML!
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out!<br/>Check it out at <a href="{{ .ReleaseURL }}">{{ .ReleaseURL }}</a>'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"
```

{% include-markdown "../../includes/templates.md" comments=false %}
