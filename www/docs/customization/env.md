# Environment Variables

Global environment variables to be passed down to all hooks and builds.

If you have an environment variable named `FOOBAR` set to `on`, your
`.goreleaser.yaml` file could use it like this:

```yaml
# .goreleaser.yaml
env:
  - FOO={{ .Env.FOOBAR }}
  - ENV_WITH_DEFAULT={{ if index .Env "ENV_WITH_DEFAULT"  }}{{ .Env.ENV_WITH_DEFAULT }}{{ else }}default_value{{ end }}
before:
  hooks:
    - go mod tidy
builds:
  - binary: program
```

This way, both your before hooks (in this example, `go mod tidy`) and the
underlying builds (using `go build`) will have `FOO` set to `on`.

The root `env` section also accepts templates.

{% include-markdown "../includes/templates.md" comments=false %}
