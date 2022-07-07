# LinkedIn

For it to work, you'll need to set some environment variables on your pipeline:

- `LINKEDIN_ACCESS_TOKEN`

!!! warning
    We currently don't support posting in groups.

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  linkedin:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # Message template to use while publishing.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}`
    message_template: 'Awesome project {{.Tag}} is out!'
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
