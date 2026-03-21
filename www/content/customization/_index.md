---
title: "Documentation"
weight: 1
---

GoReleaser is configured through a `.goreleaser.yaml` file at the root of your repository.

You can find all available options and their defaults in the sections below.

> [!NOTE]
> You can use `goreleaser init` to create a sample `.goreleaser.yaml` to get
> started, and `goreleaser check` to validate your configuration file.

## JSON Schema

GoReleaser's configuration is backed by a JSON Schema, which enables autocompletion and
validation in editors that support it.

You can instruct your LSP to use it by adding a comment to your
`.goreleaser.yaml` file:

{{< tabs >}}

{{< tab name="OSS" >}}

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
```

{{< /tab >}}
{{< tab name="Pro" >}}

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema-pro.json
```

{{< /tab >}}
{{< /tabs >}}
