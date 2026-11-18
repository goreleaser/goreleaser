---
title: "GitHub Actions"
weight: 10
---

GoReleaser can be used within the official [GoReleaser
Action][goreleaser-action] through [GitHub Actions][actions].

Create a workflow to push your releases by adding a YAML configuration
file at `.github/workflows/release.yml`.

## Usage

### Workflow

This workflow creates a release only when you push a tag:

```yaml {filename=".github/workflows/release.yml"}
name: goreleaser

on:
  push:
    # run only against tags
    tags:
      - "*"

permissions:
  contents: write
  # packages: write
  # issues: write
  # id-token: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v7
        with:
          fetch-depth: 0
      - name: Set up Go
        uses: actions/setup-go@v7
        with:
          go-version: stable
      # More assembly might be required: Docker logins, GPG, etc.
      # It all depends on your needs.
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          # either 'goreleaser' (default) or 'goreleaser-pro'
          distribution: goreleaser
          # 'latest', 'nightly', or a semver
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # Your GoReleaser Pro key, if you are using the 'goreleaser-pro' distribution
          # GORELEASER_KEY: ${{ secrets.GORELEASER_KEY }}
```

> [!WARNING]
> **Some things to look closely...**
>
> #### The action does not install, configure or authenticate into dependencies
>
> GoReleaser Action will not install nor setup any other software needed to
> release. It's the user's responsibility to install and configure Go, Docker,
> Syft, Cosign, and any other tools the release might need. It's also the
> user's responsibility to sign in to tools that need it, such as Docker.
>
> #### Fetch all history
>
> Notice the `fetch-depth: 0` option in the `Checkout` workflow step.
> This is required for GoReleaser to work properly, as it needs the full
> commit history.

> [!NOTE]
> For detailed instructions please follow GitHub Actions [workflow syntax][syntax].

### Nightly builds

Setting `version: nightly` on `goreleaser-action` (≥ `v7.2.0`) installs the
latest immutable nightly release (e.g. `v2.16.0-abc1234-nightly`) instead of a
moving tag. The action resolves it through the GitHub Releases API, so make
sure `GITHUB_TOKEN` is exported on the step (the example above already does)
to avoid the unauthenticated rate limit.

### Signing

If [signing is enabled][signing] in your GoReleaser configuration, you can use
the [Import GPG][import-gpg] GitHub Action along with this one:

```yaml {filename=".github/workflows/release.yml"}
jobs:
  # ...
  goreleaser:
    # ...
    steps:
      # ...
      - name: Import GPG key
        id: import_gpg
        uses: crazy-max/ghaction-import-gpg@v6
        with:
          gpg_private_key: ${{ secrets.GPG_PRIVATE_KEY }}
          passphrase: ${{ secrets.PASSPHRASE }}
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
      # ...
```

And reference the fingerprint in your signing configuration using the
`GPG_FINGERPRINT` environment variable:

```yaml {filename=".goreleaser.yaml"}
signs:
  - artifacts: checksum
    cmd: gpg2
    args:
      - "--batch"
      - "-u"
      - "{{ .Env.GPG_FINGERPRINT }}"
      - "--output"
      - "${signature}"
      - "--detach-sign"
      - "${artifact}"
```

## Customizing

### Inputs

The following inputs can be used as `step.with` keys:

| Name           | Type   | Default      | Description                                                      |
| -------------- | ------ | ------------ | ---------------------------------------------------------------- |
| `distribution` | String | `goreleaser` | GoReleaser distribution, either `goreleaser` or `goreleaser-pro` |
| `version`[^1]  | String | `~> v2`      | GoReleaser version                                               |
| `args`         | String |              | Arguments to pass to GoReleaser                                  |
| `workdir`      | String | `.`          | Working directory (below repository root)                        |
| `install-only` | Bool   | `false`      | Just install GoReleaser                                          |

[^1]:
    Can be a fixed version like `v0.117.0` or a max satisfying SemVer one like
    `~> 0.132`. In this case this will return `v0.132.1`.

### Outputs

The following outputs are available:

| Name        | Type | Description            |
| ----------- | ---- | ---------------------- |
| `artifacts` | JSON | Build result artifacts |
| `metadata`  | JSON | Build result metadata  |

### Environment Variables

The following environment variables can be used as `step.env` keys:

| Name             | Description                                                                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GITHUB_TOKEN`   | [GITHUB_TOKEN](https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token) as provided by `secrets` |
| `GORELEASER_KEY` | Your [GoReleaser Pro](https://goreleaser.com/pro) License Key, in case you are using the `goreleaser-pro` distribution                              |

## Job Summary

{{< g_version "v2.18" >}}

GoReleaser adds a [release summary](/customization/summary/) to the job summary
view, listing what the run published and linking to it. GitHub Actions sets the
`GITHUB_STEP_SUMMARY` environment variable it needs, so there is nothing to
configure.

## Token Permissions

The following
[permissions](https://docs.github.com/en/actions/reference/authentication-in-a-workflow#permissions-for-the-github_token)
are required by GoReleaser:

- `contents: write` if you wish to
  - upload archives as GitHub Releases, or
  - publish to [Homebrew](/customization/publish/homebrew_casks/), or
    [Scoop](/customization/publish/scoop/) (assuming it's part of the same repository)
- or just `contents: read` if you don't need any of the above
- `packages: write` if you [push Docker images](/customization/package/docker/) to
  GitHub
- `issues: write` if you use [milestone closing
  capability](/customization/publish/milestone/)
- `id-token: write` if you wish use [Cosign][cosign] with [GitHub OIDC][oidc]

`GITHUB_TOKEN` permissions [are limited to the repository][about-github-token]
that contains your workflow.

To publish to another repository, use a GitHub App installation token or a
personal access token with access to that repository.

### GitHub App token

For example, to publish a Homebrew cask and a Nix package to
`my-org/homebrew-tap` and `my-org/nur`:

1. [Create a GitHub App][create-app] with repository `Contents: read and write`
   permission and install it on both repositories.
2. Add the app's client ID as the `APP_CLIENT_ID` repository variable and its
   private key as the `APP_PRIVATE_KEY` repository secret in the repository
   that runs the workflow.
3. Generate an installation token before running GoReleaser:

```yaml {filename=".github/workflows/release.yml"}
jobs:
  # ...
  goreleaser:
    # ...
    steps:
      # ...
      - name: Create publishing token
        id: publishing-token
        uses: actions/create-github-app-token@v3
        with:
          client-id: ${{ vars.APP_CLIENT_ID }}
          private-key: ${{ secrets.APP_PRIVATE_KEY }}
          owner: my-org
          repositories: |
            homebrew-tap
            nur
          permission-contents: write
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          PUBLISH_TOKEN: ${{ steps.publishing-token.outputs.token }}
```

Configure the cask and Nix package to use the publishing token while
`GITHUB_TOKEN` remains available for the release repository:

```yaml {filename=".goreleaser.yaml"}
homebrew_casks:
  - repository:
      owner: my-org
      name: homebrew-tap
      token: "{{ .Env.PUBLISH_TOKEN }}"
    commit_author:
      use_github_app_token: true
nix:
  - repository:
      owner: my-org
      name: nur
      token: "{{ .Env.PUBLISH_TOKEN }}"
    commit_author:
      use_github_app_token: true
```

With `use_github_app_token: true` (since v2.13), GoReleaser omits the custom
committer when publishing through the GitHub API. GitHub uses the app's bot
identity as the author and committer and signs the commits. You do not need to
look up the bot's user ID or configure its name and email.

The repositories in a token must belong to the same owner. For repositories
under different owners, install the app on each owner and create a token for
each installation.

Installation tokens expire after one hour. For releases that take longer,
see the [token action documentation][app-token].

### Personal access token

Alternatively, create a [Personal Access Token][pat] with access to the
repositories GoReleaser needs and [add it as a secret][secrets]. For example,
if you create a secret named `GH_PAT`, the step will look like this:

```yaml {filename=".github/workflows/release.yml"}
jobs:
  # ...
  goreleaser:
    # ...
    steps:
      # ...
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
      # ...
```

You can also read the [GitHub documentation](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) about it.

## What does it look like?

You can check [this example repository](https://github.com/goreleaser/example/releases/latest) for a real world example.

[goreleaser-action]: https://github.com/goreleaser/goreleaser-action
[actions]: https://github.com/features/actions
[syntax]: https://help.github.com/en/articles/workflow-syntax-for-github-actions#About-yaml-syntax-for-workflows
[signing]: /customization/sign/sign/
[import-gpg]: https://github.com/crazy-max/ghaction-import-gpg
[github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token
[about-github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token#about-the-github_token-secret
[pat]: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
[secrets]: https://help.github.com/en/actions/automating-your-workflow-with-github-actions/creating-and-using-encrypted-secrets
[create-app]: https://docs.github.com/en/apps/creating-github-apps/setting-up-a-github-app/creating-a-github-app
[app-token]: https://github.com/actions/create-github-app-token
[oidc]: https://docs.github.com/en/actions/security-for-github-actions/security-hardening-your-deployments/about-security-hardening-with-openid-connect#adding-permissions-settings
[cosign]: https://github.com/sigstore/cosign
