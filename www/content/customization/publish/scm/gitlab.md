---
title: "GitLab"
weight: 20
---

> [!WARNING]
> Only GitLab `v12.9+` is supported for releases.

## Subgroups and project ID

If you use GitLab subgroups, you need to specify it in the `owner` field,
e.g. `mygroup/mysubgroup`.

You can also use Gitlab's internal project id by setting it in the
`release.gitlab.name` field and leaving the owner field empty.

## API Token

GoReleaser requires an API token with the `api` scope selected to deploy the artifacts to GitLab.
That token can either be a Personal, or a Project one.

This token should be added to the environment variables as `GITLAB_TOKEN`.

Alternatively, you can provide the GitLab token in a file.
GoReleaser will check `~/.config/goreleaser/gitlab_token` by default, but you can change that in the `.goreleaser.yaml` file:

```yaml {filename=".goreleaser.yaml"}
env_files:
  gitlab_token: ~/.path/to/my/gitlab_token
```

> [!WARNING]
> If you use a project access token, make sure to set `use_package_registry`
> to `true` as well, otherwise it might not work.

> [!WARNING]
> If you are using a [protected variable](https://docs.gitlab.com/ee/ci/variables/#protected-cicd-variables)
> to store any of the values needed by goreleaser, ensure that you are protecting the tags as CI jobs in
> Gitlab only may access protected variables if the job is run for protected refs
> ([branches](https://docs.gitlab.com/ee/user/project/protected_branches.html),
> [tags](https://docs.gitlab.com/ee/user/project/protected_tags.html)).

## GitLab Enterprise or private hosted

You can use GoReleaser with GitLab Enterprise by providing its URLs in the
`.goreleaser.yml` configuration file. This takes a normal string, or a template value.

```yaml {filename=".goreleaser.yaml"}
gitlab_urls:
  api: https://gitlab.mycompany.com/api/v4/
  download: https://gitlab.company.com

  # set to true if you use a self-signed certificate
  skip_tls_verify: false

  # set to true if you want to upload to the Package Registry rather than attachments
  # Only works with GitLab 13.5+
  use_package_registry: false

  # Set this if you set GITLAB_TOKEN to the value of CI_JOB_TOKEN.
  use_job_token: true
```

If none are set, they default to GitLab's public URLs.

> [!NOTE]
> Releasing to a private-hosted GitLab CE will only work for version `v12.9+`, due to dependencies
> on [release](https://docs.gitlab.com/ee/user/project/releases/index.html) functionality
> and [direct asset linking](https://docs.gitlab.com/ee/user/project/releases/index.html#permanent-links-to-release-assets).

## Generic Package Registry

GitLab introduced the [Generic Package Registry](https://docs.gitlab.com/ee/user/packages/package_registry/index.html) in Gitlab 13.5.

Normally, `goreleaser` uploads release files as "attachments", which may have [administrative limits](https://docs.gitlab.com/ee/administration/settings/account_and_limit_settings.html).
Notably, hosted GitLab instances have a 10MB attachment limit, which cannot be changed.

Uploading to the Generic Package Registry does not have this restriction.
To use it instead, set `use_package_registry` to `true`.

```yaml {filename=".goreleaser.yaml"}
gitlab_urls:
  use_package_registry: true
```

## Direct asset URLs

By default, GoReleaser asks GitLab to create a permanent direct asset link for
each uploaded artifact.

Some older GitLab instances may create broken direct asset URLs when GoReleaser
sends an asset path for GitLab to derive the final URL. This has been observed
on GitLab Community Edition 13.4.3 and is tracked in
[goreleaser/goreleaser#3299](https://github.com/goreleaser/goreleaser/issues/3299).
If your release links work, but the generated direct asset URLs return 404, set
`use_direct_asset_url` to `true`:

```yaml {filename=".goreleaser.yaml"}
gitlab_urls:
  use_direct_asset_url: true
```

With this option enabled, GoReleaser sends the uploaded artifact URL as the
release link's direct asset URL instead of asking GitLab to derive it from an
asset path. Keep the default value unless your GitLab version accepts
`direct_asset_url` in release link create requests and you need this
compatibility behavior.

## Example release

You can check [this example repository](https://gitlab.com/goreleaser/example/-/releases) for a real world example.
