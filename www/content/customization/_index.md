---
title: "Introduction"
weight: 1
---

GoReleaser can be customized by tweaking a `.goreleaser.yaml`[^goreleaser-yaml]
file.

Run `goreleaser init` to generate an example config, or start from scratch.

Run `goreleaser check` to verify your config: it tells you whether you're using
deprecated or invalid options.

## JSON Schema

GoReleaser publishes a [JSON Schema][jsonschema] file, which you can use to get
better editor support:

{{< tabs >}}

{{< tab name="OSS" >}}

```sh
https://goreleaser.com/static/schema.json
```

You can also reference it from your config file by adding a comment like the
following:

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
```

{{< /tab >}}
{{< tab name="Pro" >}}

```sh
https://goreleaser.com/static/schema-pro.json
```

You can also reference it from your config file by adding a comment like the
following:

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema-pro.json
```

{{< /tab >}}
{{< /tabs >}}

To generate the schema for the exact version you run, use the
`goreleaser jsonschema` command.

### Pin the schema version

You can pin the version by getting the schema from the GitHub tag, for example,
for `__VERSION__` (latest):

{{< tabs >}}

{{< tab name="OSS" >}}

```sh
https://raw.githubusercontent.com/goreleaser/goreleaser/__VERSION__/www/static/schema.json
```

{{< /tab >}}
{{< tab name="Pro" >}}

```sh
https://raw.githubusercontent.com/goreleaser/goreleaser/__VERSION__/www/static/schema-pro.json
```

{{< /tab >}}
{{< /tabs >}}

[^goreleaser-yaml]:
    While most of the documentation refers to the `.goreleaser.yaml` filename
    for simplicity, a few different variants of it are actually accepted.
    In order of precedence:

    - `.config/goreleaser.yml`
    - `.config/goreleaser.yaml`
    - `.goreleaser.yml`
    - `.goreleaser.yaml`
    - `goreleaser.yml`
    - `goreleaser.yaml`

[jsonschema]: https://json-schema.org/draft/2020-12/json-schema-validation.html
