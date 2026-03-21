---
title: "GitHub"
weight: 10
---

## API Token

GoReleaser requires an API token with the `repo` scope selected to deploy the
artifacts to GitHub. You can create one
[here](https://github.com/settings/tokens/new).

This token should be added to the environment variables as `GITHUB_TOKEN`.

Alternatively, you can provide the GitHub token in a file. GoReleaser will check
`~/.config/goreleaser/github_token` by default, but you can change that in the
`.goreleaser.yaml` file:

```yaml {filename=".goreleaser.yaml"}
env_files:
  github_token: ~/.path/to/my/github_token
```

Note that the environment variable will be used if available, regardless of the
`github_token` file.

## GitHub Enterprise

You can use GoReleaser with GitHub Enterprise by providing its URLs in the
`.goreleaser.yaml` configuration file. This takes a normal string, or a template
value.

```yaml {filename=".goreleaser.yaml"}
github_urls:
  api: https://git.company.com/api/v3/
  upload: https://git.company.com/api/uploads/
  download: https://git.company.com/
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

If none are set, they default to GitHub's public URLs.

## Example release

You can check [this example repository](https://github.com/goreleaser/example/releases/latest) for a real world example.
