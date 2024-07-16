# SMTP

For it to work, you'll need to set some environment variables on your pipeline:

- `SMTP_PASSWORD`

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml
# .goreleaser.yaml
announce:
  smtp:
    # Whether its enabled or not.
    enabled: true

    # SMTP Host.
    #
    # Default: '$SMTP_HOST'.
    host: "smtp.gmail.com"

    # SMTP Port
    #
    # Default: '$SMTP_PORT'.
    port: 587

    # Sender of the email
    from: ""

    # Receivers of the email
    to:
      - ""
      - ""

    # Owner of the email
    #
    # Default: '$SMTP_USERNAME'.
    username: ""

    # Body to use within the email.
    #
    # Default: 'You can view details from: {{ .ReleaseURL }}'.
    # Templates: allowed.
    body_template: "https://github.com/goreleaser/goreleaser/releases/tag/{{ .Tag }}"

    # Subject template to use within the email subject.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out!'.
    # Templates: allowed.
    subject_template: "GoReleaser {{ .Tag }} was just released!"
```

{% include-markdown "../../includes/templates.md" comments=false %}
