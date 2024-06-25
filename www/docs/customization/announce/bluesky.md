# Bluesky

To use [Bluesky](https://bsky.app/), you need to create an account, and set the
following environment variable on your pipeline:

- `BLUESKY_APP_PASSWORD` (create one [here](https://bsky.app/settings/app-passwords))

After this, you can add following section to your `.goreleaser.yaml`
configuration:

```yaml
# .goreleaser.yaml
announce:
  bluesky:
    # Whether it's enabled or not.
    enabled: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"

    # The username of the account that will post
    # to Bluesky
    username: "my-project.bsky.social"
```

{% include-markdown "../../includes/templates.md" comments=false %}
