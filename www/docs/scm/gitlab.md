# GitLab

## API Token

GoReleaser requires an API token with the `api` scope selected to deploy the artifacts to GitLab.
You can create one [here](https://gitlab.com/profile/personal_access_tokens).

This token should be added to the environment variables as `GITLAB_TOKEN`.

Alternatively, you can provide the GitLab token in a file.
GoReleaser will check `~/.config/goreleaser/gitlab_token` by default, but you can change that in the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
env_files:
  gitlab_token: ~/.path/to/my/gitlab_token
```

## GitLab Enterprise or private hosted

You can use GoReleaser with GitLab Enterprise by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
gitlab_urls:
  api: https://gitlab.mycompany.com/api/v4/
  download: https://gitlab.company.com
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```

If none are set, they default to GitLab's public URLs.

!!! note
    Releasing to a private-hosted GitLab CE will only work for version `v12.9+`, due to dependencies
    on [release](https://docs.gitlab.com/ee/user/project/releases/index.html) functionality
    and [direct asset linking](https://docs.gitlab.com/ee/user/project/releases/index.html#permanent-links-to-release-assets).


## Example release

Here's an example of how the release might look like:

<a href="https://gitlab.com/goreleaser/example/-/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-gitlab.png"/>
    <figcaption>Example release on GitLab.</figcaption>
  </figure>
</a>
