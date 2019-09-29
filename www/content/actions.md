---
title: GitHub Actions
menu: true
weight: 141
---

GoReleaser can also be used within our official [GoReleaser Action][goreleaser-action] through [GitHub Actions][actions].

You can create a workflow for pushing your releases by putting YAML configuration to `.github/workflows/release.yml`.

Below is a simple snippet to use this action in your workflow:
```yaml
name: goreleaser

on:
  pull_request:
  push:

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      -
        name: Checkout
        uses: actions/checkout@master
      -
        name: Set up Go
        uses: actions/setup-go@master
      -
        name: Run GoReleaser
        uses: goreleaser/goreleaser-action@master
        with:
          version: latest
          args: release
```

> For detailed intructions please follow GitHub Actions [workflow syntax][syntax].

## Customizing

### Inputs

Following inputs can be used as `step.with` keys

| Name          | Type    | Default   | Description                              |
|---------------|---------|-----------|------------------------------------------|
| `version`     | String  | `latest`  | GoReleaser version. Example: `v0.117.0`  |
| `args`        | String  |           | Arguments to pass to GoReleaser          |
| `key`         | String  |           | Private key to import                    |

### Signing

If signing is enabled in your GoReleaser configuration, populate the
`key` input with your private key and reference the key in your signing
configuration, e.g.

```yaml
signs:
  - artifacts: checksum
    args: ["--batch", "-u", "<key id, fingerprint, email, ...>", "--output", "${signature}", "--detach-sign", "${artifact}"]
```

This feature is currently only compatible when using the default `gpg`
command and a private key without a passphrase.

[goreleaser-action]: https://github.com/goreleaser/goreleaser-action
[actions]: https://github.com/features/actions
[syntax]: https://help.github.com/en/articles/workflow-syntax-for-github-actions#About-yaml-syntax-for-workflows
