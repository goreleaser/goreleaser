# WebHooks

WebHooks are a way to receive notifications.
With this GoReleaser functionality, you can send events to any server
exposing a WebHook.

If your endpoints are not secure, you can use following environment variables to configure them:

- `BASIC_AUTH_HEADER_VALUE` like `Basic <base64(username:password)>`
- `BEARER_TOKEN_HEADER_VALUE` like `Bearer <token>`

Add following to your `.goreleaser.yaml` configuration to enable the WebHook functionality:

```yaml title=".goreleaser.yaml"
announce:
  webhook:
    # Whether its enabled or not.
    #
    # Templates: allowed (since v2.6).
    enabled: true

    # Check the certificate of the webhook.
    skip_tls_verify: true

    # Message template to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: '{ "title": "Awesome project {{.Tag}} is out!"}'

    # Content type to use.
    #
    # Default: 'application/json; charset=utf-8'.
    content_type: "application/json"

    # Endpoint to send the webhook to.
    endpoint_url: "https://example.com/webhook"
    # Headers to send with the webhook.
    # For example:
    # headers:
    #   Authorization: "Bearer <token>"
    headers:
      User-Agent: "goreleaser"

    # HTTP status codes to be considered as a successful response.
    #
    # Default: 200, 201, 202, 204
    expected_status_codes: [418, 200, 201]
```

<!-- md:templates -->
