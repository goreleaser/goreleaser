# SMTP

For it to work, you'll need to set some environment variables on your pipeline:

- `SMTP_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yml` config:

```yaml
# .goreleaser.yml
announce:
  smtp:
    # Whether its enabled or not.
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
    # Defaults to `You can view details from: {{ trimsuffix .GitURL ".git" }}/releases/tag/{{ .Tag }}`
    body_template: 'https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}'

    # Subject template to use within the email subject.
    # Defaults to `{{ .ProjectName }} {{ .Tag }} is out!`
    subject_template: ''GoReleaser {{ .Tag }} was just released!''
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).
