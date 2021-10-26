# Customization

GoReleaser can be customized by tweaking a `.goreleaser.yml` file.

You can generate an example config by running [`goreleaser init`](/cmd/goreleaser_init/) or start from scratch.

You can also check if your config is valid by running [`goreleaser check`](/cmd/goreleaser_check/), which will tell you if are using deprecated or invalid options.

## JSON Schema

GoReleaser also has a [jsonschema][] file which you can use to have better editor support:

=== "OSS"
    ```sh
    https://goreleaser.com/schema.json
    ```

=== "Pro"
    ```sh
    https://goreleaser.com/schema-pro.json
    ```

You can also generate it for your specific version using the [`goreleaser jsonschema`][schema] command.

[jsonschema]: http://json-schema.org/draft/2020-12/json-schema-validation.html
[schema]: /cmd/goreleaser_jsonschema/
