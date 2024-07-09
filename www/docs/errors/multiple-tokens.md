# Multiple tokens found, but only one is allowed

GoReleaser infers if you are using GitHub, GitLab or Gitea by which tokens are provided.
If you have multiple tokens set, you'll get this error.

Here's an example:

```sh
   ⨯ release failed after 0.02s error=gmultiple tokens found, but only one is allowed: GITHUB_TOKEN, GITLAB_TOKEN

Learn more at https://goreleaser.com/errors/multiple-tokens
```

In this case, you either unset `GITHUB_TOKEN` or `GITLAB_TOKEN`.
You can read more about it in the [SCM docs](../scm/github.md).

This can also happen if you load the tokens from files.
The default paths are:

- `~/.config/goreleaser/github_token`
- `~/.config/goreleaser/gitlab_token`
- `~/.config/goreleaser/gitea_token`

If you have more than one of these files, but for a particular project, you want
to force one of them, you can explicitly disable the others by setting them to a
file you know won't exist:

```yaml
# .goreleaser.yaml
env_files:
  gitlab_token: ~/nope
  gitea_token: ~/nope
```

This will prevent using both GitLab and Gitea tokens.

## Forcing a specific token

If GoReleaser is being run with more than one of the `*_TOKEN` environment
variables and you can't unset any of them, you can force GoReleaser to use a
specific one by exporting a `GORELEASER_FORCE_TOKEN` environment variable.

So, for instance, if you have both `GITHUB_TOKEN` and `GITEA_TOKEN` set and want
GoReleaser to pick `GITEA_TOKEN`, you can set `GORELEASER_FORCE_TOKEN=gitea`.
GoReleaser will then unset `GITHUB_TOKEN` and proceed.

You can also force a token by using `force_token` in your config:

```yaml
# .goreleaser.yaml
force_token: gitea
```
