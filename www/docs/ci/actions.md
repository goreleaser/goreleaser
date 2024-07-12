# GitHub Actions

GoReleaser can also be used within our official [GoReleaser
Action][goreleaser-action] through [GitHub Actions][actions].

You can create a workflow for pushing your releases by putting YAML
configuration to `.github/workflows/release.yml`.

## Usage

### Workflow

Below is a simple snippet to use this action in your workflow:

```yaml
# .github/workflows/release.yml
name: goreleaser

on:
  pull_request:
  push:
    # run only against tags
    tags:
      - "*"

permissions:
  contents: write
  # packages: write
  # issues: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: stable
      # More assembly might be required: Docker logins, GPG, etc.
      # It all depends on your needs.
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          # either 'goreleaser' (default) or 'goreleaser-pro'
          distribution: goreleaser
          # 'latest', 'nightly', or a semver
          version: "~> v1"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          # Your GoReleaser Pro key, if you are using the 'goreleaser-pro' distribution
          # GORELEASER_KEY: ${{ secrets.GORELEASER_KEY }}
```

!!! warning "Some things to look closely..."

    #### The action does not install, configure or authenticate into dependencies
    GoReleaser Action will not install nor setup any other software needed to
    release. It's the user's responsibility to install and configure Go, Docker,
    Syft, Cosign and any other tools the release might need. It's also the
    user's responsibility to log in into tools that need it, such as docker.

    #### Fetch depthness
    Notice the `fetch-depth: 0` option on the `Checkout` workflow step.
    It is required for GoReleaser to work properly.
    Without that, GoReleaser might fail or behave incorrectly.

    #### Tag fetching
    Notice the `git fetch --force -tags`. This is needed if you use fields like
    `TagBody`, `TagSubject` or `TagContents` in your templates.
    For more information, take a look at
    [actions/checkout#290](https://github.com/actions/checkout/issues/290).

!!! tip

    For detailed instructions please follow GitHub Actions [workflow syntax][syntax].

### Signing

If [signing is enabled][signing] in your GoReleaser configuration, you can use
the [Import GPG][import-gpg] GitHub Action along with this one:

```yaml
# .github/workflows/release.yml
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
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v1"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GPG_FINGERPRINT: ${{ steps.import_gpg.outputs.fingerprint }}
      # ...
```

And reference the fingerprint in your signing configuration using the
`GPG_FINGERPRINT` environment variable:

```yaml
# .goreleaser.yaml
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

Following inputs can be used as `step.with` keys

| Name           | Type   | Default      | Description                                                      |
| -------------- | ------ | ------------ | ---------------------------------------------------------------- |
| `distribution` | String | `goreleaser` | GoReleaser distribution, either `goreleaser` or `goreleaser-pro` |
| `version`[^1]  | String | `~> v1`      | GoReleaser version                                               |
| `args`         | String |              | Arguments to pass to GoReleaser                                  |
| `workdir`      | String | `.`          | Working directory (below repository root)                        |
| `install-only` | Bool   | `false`      | Just install GoReleaser                                          |

[^1]:
    Can be a fixed version like `v0.117.0` or a max satisfying SemVer one like
    `~> 0.132`. In this case this will return `v0.132.1`.

### Outputs

Following outputs are available

| Name        | Type | Description            |
| ----------- | ---- | ---------------------- |
| `artifacts` | JSON | Build result artifacts |
| `metadata`  | JSON | Build result metadata  |

### Environment Variables

Following environment variables can be used as `step.env` keys

| Name             | Description                                                                                                                                         |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GITHUB_TOKEN`   | [GITHUB_TOKEN](https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token) as provided by `secrets` |
| `GORELEASER_KEY` | Your [GoReleaser Pro](https://goreleaser.com/pro) License Key, in case you are using the `goreleaser-pro` distribution                              |

## Token Permissions

The following
[permissions](https://docs.github.com/en/actions/reference/authentication-in-a-workflow#permissions-for-the-github_token)
are required by GoReleaser:

- `contents: write` if you wish to
  - [upload archives as GitHub Releases](../customization/release.md), or
  - publish to [Homebrew](../customization/homebrew.md), or
    [Scoop](../customization/scoop.md) (assuming it's part of the same repository)
- or just `contents: read` if you don't need any of the above
- `packages: write` if you [push Docker images](../customization/docker.md) to
  GitHub
- `issues: write` if you use [milestone closing
  capability](../customization/milestone.md)

`GITHUB_TOKEN` permissions [are limited to the repository][about-github-token]
that contains your workflow.

If you need to push the homebrew tap to another repository, you must create a
custom [Personal Access Token][pat] with `repo` permissions and [add it as a
secret in the repository][secrets]. If you create a secret named `GH_PAT`, the
step will look like this:

```yaml
# .github/workflows/release.yml
jobs:
  # ...
  goreleaser:
    # ...
    steps:
      # ...
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v1"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
      # ...
```

You can also read the [GitHub documentation](https://docs.github.com/en/github/authenticating-to-github/creating-a-personal-access-token) about it.

## What does it look like?

You can check [this example repository](https://github.com/goreleaser/example) for a real world example.

<a href="https://github.com/goreleaser/example/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-github.png"/>
    <figcaption>Example release on GitHub.</figcaption>
  </figure>
</a>

[goreleaser-action]: https://github.com/goreleaser/goreleaser-action
[actions]: https://github.com/features/actions
[syntax]: https://help.github.com/en/articles/workflow-syntax-for-github-actions#About-yaml-syntax-for-workflows
[signing]: ../customization/sign.md
[import-gpg]: https://github.com/crazy-max/ghaction-import-gpg
[github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token
[about-github-token]: https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token#about-the-github_token-secret
[pat]: https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/
[secrets]: https://help.github.com/en/actions/automating-your-workflow-with-github-actions/creating-and-using-encrypted-secrets
