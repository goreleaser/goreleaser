---
title: "LinkedIn"
weight: 40
---

For it to work, you'll need to set some environment variables on your pipeline:

- `LINKEDIN_ACCESS_TOKEN`

> [!WARNING]
> We currently don't support posting in groups.

Then, you can add something like the following to your `.goreleaser.yaml` config:

```yaml {filename=".goreleaser.yaml"}
announce:
  linkedin:
    # Whether its enabled or not.
    #
    # Templates: allowed. {{< inline_version "v2.6" >}}
    enabled: true

    # Message to use while publishing.
    #
    # Default: '{{ .ProjectName }} {{ .Tag }} is out! Check it out at {{ .ReleaseURL }}'.
    message_template: "Awesome project {{.Tag}} is out!"
```

{{< templates >}}
