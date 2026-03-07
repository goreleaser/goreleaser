# Discourse

<!-- md:version v2.13 -->

This announcer enables posting new release messages to a
[Discourse](https://discourse.org/) forum.
It will create a new Discourse "topic" (a new post/thread) in the desired
category.

## Setup

To setup, a forum admin will need to create an API key at
`https://<your.forum.hostname>/admin/api/keys`.
While not required, for security the recommended settings are:

- User level -> Single user
- Scope -> Granular
  - `(x) topics/write`

Where GoReleaser is running, the following environment variable should be set
with the API key as the value:

- `DISCOURSE_API_KEY`

After this, you can add following section to your `.goreleaser.yaml`
configuration:

```yaml title=".goreleaser.yaml"
announce:
  discourse:
    # Whether this announcer is enabled or not.
    #
    # Templates: allowed.
    enabled: true

    # The fully qualified domain name (FQDN) of the Discourse forum.
    # Do not include a trailing slash.
    #
    # Required.
    server: my.forum.com

    # Title to use for the Discourse topic.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out!'.
    # Templates: allowed.
    title_template: "GoReleaser {{ .Tag }} was just released!"

    # Message to use in the post body.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    # Templates: allowed.
    message_template: "Awesome project {{.Tag}} is out!"

    # The Discourse username that will be author of this topic. Needs to be an
    # existing username. `system` is the built-in Discourse username.

    # Default: system
    username: "GoReleaser"

    # The Discourse category id to post to. Needs to be an integer. You can
    # find a category's ID in the browser URL when viewing a category.
    category_id: 4
```

<!-- md:templates -->

## Troubleshooting

If you get the error message:

```text
discourse: There was an error posting to Discourse. Check your config again. HTTP code: XXX
```

Then double check the Discourse section of your GoReleaser configuration.
Make sure everything is correct.
Here are some common error codes and what they **might** mean:

- 404 - The server field is either incorrect or your forum is down/unreachable.
- 403 - The API Key doesn't have the correct permission it needs or the
  `username` key in GoReleaser configuration doesn't match what's configured in
  Discourse.
- 5XX - The Discourse forum is having a bad day and throwing back errors.
