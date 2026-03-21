---
title: "Documentation"
weight: 1
---

GoReleaser is configured through a `.goreleaser.yaml` file at the root of your repository.

You can find all available options and their defaults in the sections below.

> [!NOTE]
> You can use `goreleaser init` to create a sample `.goreleaser.yaml` to get started, and
> `goreleaser check` to validate your configuration file.

## JSON Schema

GoReleaser's configuration is backed by a JSON Schema, which enables autocompletion and
validation in editors that support it. The schema is available at
`https://goreleaser.com/static/schema.json`.

You can add it to your `.goreleaser.yaml` with:

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
```
