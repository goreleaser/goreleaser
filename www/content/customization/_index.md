---
title: "Introduction"
weight: 1
---

GoReleaser can be customized by tweaking a `.goreleaser.yaml`[^goreleaser-yaml] file.

You can generate an example config by running
`goreleaser init` or start from scratch.

You can also check if your config is valid by running
`goreleaser check`, which will tell you if are
using deprecated or invalid options.

## JSON Schema

GoReleaser also has a [jsonschema][] file, which you can use to have better
editor support:

{{< tabs >}}

{{< tab name="OSS" >}}

```sh
https://goreleaser.com/static/schema.json
```

You can also specify it in your `.goreleaser.yml` config file by adding a
comment like the following:

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
```

{{< /tab >}}
{{< tab name="Pro" >}}

```sh
https://goreleaser.com/static/schema-pro.json
```

You can also specify it in your `.goreleaser.yml` config file by adding a
comment like the following:

```yaml {filename=".goreleaser.yaml"}
# yaml-language-server: $schema=https://goreleaser.com/static/schema-pro.json
```

{{< /tab >}}
{{< /tabs >}}

You can also generate it for your specific version using the
`goreleaser jsonschema` command.

### Pin the schema version

You can pin the version by getting the schema from the GitHub tag, for example,
for `__VERSION__` (latest):

{{< tabs >}}

{{< tab name="OSS" >}}

```sh
https://raw.githubusercontent.com/goreleaser/goreleaser/__VERSION__/www/docs/static/schema.json
```

{{< /tab >}}
{{< tab name="Pro" >}}

```sh
https://raw.githubusercontent.com/goreleaser/goreleaser/__VERSION__/www/docs/static/schema-pro.json
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

[jsonschema]: http://json-schema.org/draft/2020-12/json-schema-validation.html
