# GitHub

## API Token

GoReleaser requires an API token with the `repo` scope selected to deploy the artifacts to GitHub.
You can create one [here](https://github.com/settings/tokens/new).

This token should be added to the environment variables as `GITHUB_TOKEN`.

Alternatively, you can provide the GitHub token in a file.
GoReleaser will check `~/.config/goreleaser/github_token` by default, but you can change that in the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
env_files:
  github_token: ~/.path/to/my/github_token
```

## GitHub Enterprise

You can use GoReleaser with GitHub Enterprise by providing its URLs in the
`.goreleaser.yml` configuration file. This takes a normal string or a template
value.

```yaml
# .goreleaser.yml
github_urls:
  api: https://git.company.com/api/v3/
  upload: https://git.company.com/api/uploads/
  download: https://git.company.com/
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

If none are set, they default to GitHub's public URLs.

## Example release

Here's an example of how the release might look like:

<a href="https://github.com/goreleaser/example/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-github.png"/>
    <figcaption>Example release on GitHub.</figcaption>
  </figure>
</a>
