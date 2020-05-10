---
title: GitHub Actions
menu: true
weight: 141
---

GoReleaser can also be used within our official [GoReleaser Action][goreleaser-action]
through [GitHub Actions][actions].

You can create a workflow for pushing your releases by putting YAML configuration to
`.github/workflows/release.yml`.

## Usage

### Workflow

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
        uses: actions/checkout@v2
      -
        name: Unshallow
        run: git fetch --prune --unshallow
      -
        name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.14
      -
        name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v2
        with:
          version: latest
          args: release --rm-dist
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

!!! info
    Note the `Unshallow` workflow step. It is required for the changelog to work correctly.

### Run on new tag

If you want to run GoReleaser only on new tag, you can use this event:

```yaml
on:
  push:
    tags:
      - '*'
```

Or with a condition on GoReleaser step:

```yaml
      -
        name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v2
        if: startsWith(github.ref, 'refs/tags/')
        with:
          version: latest
          args: release --rm-dist
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

> For detailed instructions please follow GitHub Actions [workflow syntax][syntax].

### Signing

If [signing is enabled][signing] in your GoReleaser configuration, you can use the [Import GPG][import-gpg]
GitHub Action along with this one:

```yaml
      -
        name: Import GPG key
        id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v1
        env:
          GPG_PRIVATE_KEY: ${{ secrets.GPG_PRIVATE_KEY }}
          PASSPHRASE: ${{ secrets.PASSPHRASE }}
      -
        name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v2
        with:
          version: latest
          args: release --rm-dist
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
```

And reference the fingerprint in your signing configuration using the `GPG_FINGERPRINT` environment variable:

```yaml
signs:
  - artifacts: checksum
    args: ["--batch", "-u", "{{ .Env.GPG_FINGERPRINT }}", "--output", "${signature}", "--detach-sign", "${artifact}"]
```

## Customizing

### inputs

Following inputs can be used as `step.with` keys

| Name          | Type    | Default   | Description                               |
|---------------|---------|-----------|-------------------------------------------|
| `version`     | String  | `latest`  | GoReleaser version. Example: `v0.117.0`   |
| `args`        | String  |           | Arguments to pass to GoReleaser           |
| `workdir`     | String  | `.`       | Working directory (below repository root) |

### environment variables

Following environment variables can be used as `step.env` keys

| Name           | Description                                           |
|----------------|-------------------------------------------------------|
| `GITHUB_TOKEN` | [GITHUB_TOKEN][github-token] as provided by `secrets` |

## Limitation

`GITHUB_TOKEN` permissions [are limited to the repository][about-github-token] that contains your workflow.

If you need to push the homebrew tap to another repository, you must therefore create a custom
[Personal Access Token][pat] with `repo` permissions and [add it as a secret in the repository][secrets]. If you
create a secret named `GH_PAT`, the step will look like this:

```yaml
      -
        name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v2
        with:
          version: latest
          args: release --rm-dist
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
```

[goreleaser-action]: https://github.com/goreleaser/goreleaser-action
[actions]: https://github.com/features/actions
[syntax]: https://help.github.com/en/articles/workflow-syntax-for-github-actions#About-yaml-syntax-for-workflows
[signing]: https://goreleaser.com/customization/#Signing
[import-gpg]: https://github.com/crazy-max/ghaction-import-gpg
[github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token
[about-github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token#about-the-github_token-secret
[pat]: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
[secrets]: https://help.github.com/en/actions/automating-your-workflow-with-github-actions/creating-and-using-encrypted-secrets
