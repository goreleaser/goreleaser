# GitLab

## API Token

GoReleaser requires an API token with the `api` scope selected to deploy the artifacts to GitLab.
That token can either be a Personal, or a Project one.

This token should be added to the environment variables as `GITLAB_TOKEN`.

Alternatively, you can provide the GitLab token in a file.
GoReleaser will check `~/.config/goreleaser/gitlab_token` by default, but you can change that in the `.goreleaser.yaml` file:

```yaml
# .goreleaser.yaml
env_files:
  gitlab_token: ~/.path/to/my/gitlab_token
```

!!! warning
    If you use a project access token, make sure to set `use_package_registry`
    to `true` as well, otherwise it might not work.

## GitLab Enterprise or private hosted

You can use GoReleaser with GitLab Enterprise by providing its URLs in the
`.goreleaser.yml` configuration file. This takes a normal string, or a template value.

```yaml
# .goreleaser.yml
gitlab_urls:
  api: https://gitlab.mycompany.com/api/v4/
  download: https://gitlab.company.com

  # set to true if you use a self-signed certificate
  skip_tls_verify: false

  # set to true if you want to upload to the Package Registry rather than attachments
  # Only works with GitLab 13.5+
  # Since: v1.3.
  use_package_registry: false

  # Set this if you set GITLAB_TOKEN to the value of CI_JOB_TOKEN.
  # Default: false
  # Since: v1.11.
  use_job_token: true
```

If none are set, they default to GitLab's public URLs.

!!! note
    Releasing to a private-hosted GitLab CE will only work for version `v12.9+`, due to dependencies
    on [release](https://docs.gitlab.com/ee/user/project/releases/index.html) functionality
    and [direct asset linking](https://docs.gitlab.com/ee/user/project/releases/index.html#permanent-links-to-release-assets).

## Generic Package Registry

GitLab introduced the [Generic Package Registry](https://docs.gitlab.com/ee/user/packages/package_registry/index.html) in Gitlab 13.5.

Normally, `goreleaser` uploads release files as "attachments", which may have [administrative limits](https://docs.gitlab.com/ee/user/admin_area/settings/account_and_limit_settings.html).  Notably, hosted gitlab.com instances have a 10MB attachment limit, which cannot be changed.

Uploading to the Generic Package Registry does not have this restriction.  To use it instead, set `use_package_registry` to `true`.

```yaml
# .goreleaser.yml
gitlab_urls:
  use_package_registry: true
```

## Example release

Here's an example of what the release might look like:

<a href="https://gitlab.com/goreleaser/example/-/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-gitlab.png"/>
    <figcaption>Example release on GitLab.</figcaption>
  </figure>
</a>
