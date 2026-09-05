---
title: "Discourse"
weight: 30
---

{{< g_version "v2.13" >}}

This announcer enables posting new release messages to a
[Discourse](https://discourse.org/) forum.
It will create a new Discourse "topic" (a new post/thread) in the desired
category.

## Setup

To set up this announcer, a forum admin needs to create an API key at
`https://<your.forum.hostname>/admin/api/keys`.
While not required, these settings are recommended for security:

- User level -> Single user
- Scope -> Granular
  - `(x) topics/write`

On the machine where GoReleaser runs, set the following environment variable to
the API key:

- `DISCOURSE_API_KEY`

After this, you can add the following section to your `.goreleaser.yaml`
configuration:

```yaml {filename=".goreleaser.yaml"}
announce:
  discourse:
    # Whether this announcer is enabled or not.
    #
    # Templates: allowed.
    enabled: true

    # The absolute HTTP(S) base URL of the Discourse forum.
    # Do not include a trailing slash.
    #
    # Required.
    server: https://my.forum.com

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

{{< g_templates >}}

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
- 5XX - The Discourse forum is returning server errors.
