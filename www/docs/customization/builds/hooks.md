# Build Hooks

Both pre and post hooks are run **for each build target**, regardless of whether
these targets are generated via a matrix of OSes and architectures or defined
explicitly in `targets`, regardless of `builder`.

In addition to simple declarations as shown above _multiple_ hooks can be
declared to help retaining reusability of configuration between different build
environments.

```yaml title=".goreleaser.yaml"
builds:
  - id: "with-hooks"
    builder: go
    targets:
      - "darwin_amd64"
      - "windows_amd64"
    hooks:
      pre:
        - first-script.sh
        - second-script.sh
      post:
        - upx "{{ .Path }}"
        - codesign -project="{{ .ProjectName }}" "{{ .Path }}"
```

Each hook can also have its own work directory and environment variables:

```yaml title=".goreleaser.yaml"
builds:
  - id: "with-hooks"
    builder: go
    targets:
      - "darwin_amd64"
      - "windows_amd64"
    hooks:
      pre:
        - cmd: first-script.sh
          dir:
            "{{ dir .Dist}}"
            # Always print command output, otherwise only visible in debug mode.
          output: true
          env:
            - HOOK_SPECIFIC_VAR={{ .Env.GLOBAL_VAR }}
        - second-script.sh
```

All properties of a hook (`cmd`, `dir` and `env`) support
[templating](../templates.md) with `post` hooks having binary artifact
available (as these run _after_ the build).
Additionally the following build details are exposed to both `pre` and `post`
hooks:

| Key     | Description                            |
| ------- | -------------------------------------- |
| .Name   | Filename of the binary, e.g. `bin.exe` |
| .Ext    | Extension, e.g. `.exe`                 |
| .Path   | Absolute path to the binary            |
| .Target | Build target, e.g. `darwin_amd64`      |

Environment variables are inherited and overridden in the following order:

- global (`env`)
- build (`builds[].env`)
- hook (`builds[].hooks.pre[].env` and `builds[].hooks.post[].env`)

<!-- md:templates -->
