# SMTP

For it to work, you'll need to set some environment variables on your pipeline:

- `SMTP_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  smtp:
    # Whether its enabled or not.
    # Defaults to false.
    enabled: true

    # SMTP Host.
    # Default from $SMTP_HOST.
    host: "smtp.gmail.com"

    # SMTP Port
    # Default from $SMTP_PORT.
    port: 587

    # Sender of the email
    from: ""

    # Receivers of the email
    to:
      - ""
      - ""

    # Owner of the email
    # Default from $SMTP_USERNAME.
    username: ""

    # Body template to use within the email.
    # Defaults to `You can view details from: {{ .ReleaseURL }}`
    body_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Subject template to use within the email subject.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    subject_template: ''GoReleaser {{ .Tag }} was just released!''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
