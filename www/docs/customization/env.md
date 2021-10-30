# Environment Variables

Global environment variables to be passed down to all hooks and builds.

This is useful for `GO111MODULE`, for example. You can have your `.goreleaser.yml` file like the following:

```yaml
# .goreleaser.yml
env:
  - GO111MODULE=on
  - FOO={{ .Env.FOOBAR }}
  - ENV_WITH_DEFAULT={{ if index .Env "ENV_WITH_DEFAULT"  }}{{ .Env.ENV_WITH_DEFAULT }}{{ else }}default_value{{ end }}
before:
  hooks:
    - go mod tidy
builds:
- binary: program
```

This way, both `go mod tidy` and the underlying `go build` will have
`GO111MODULE` set to `on`.

The root `env` section also accepts templates.

!!! tip
    Learn more about the [name template engine](/customization/templates/).
