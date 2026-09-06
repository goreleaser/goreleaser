---
title: Build Hooks
weight: 100
---

Both pre and post hooks run **for each build target**, whether the targets come
from a matrix of operating systems and architectures or are listed explicitly in
`targets`, and regardless of the `builder`.

In addition to simple declarations, you can declare _multiple_ hooks to reuse
configuration between different build environments.

```yaml {filename=".goreleaser.yaml"}
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

```yaml {filename=".goreleaser.yaml"}
builds:
  - id: "with-hooks"
    builder: go
    targets:
      - "darwin_amd64"
      - "windows_amd64"
    hooks:
      pre:
        - cmd: first-script.sh
          dir: .
          # Always print command output, otherwise only visible in debug mode.
          output: true
          env:
            - HOOK_SPECIFIC_VAR={{ .Env.GLOBAL_VAR }}
        - second-script.sh
```

All properties of a hook (`cmd`, `dir` and `env`) support
[templating](/customization/general/templates/) with `post` hooks having the
binary artifact available (as these run _after_ the build).
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

{{< g_templates >}}
