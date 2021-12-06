# Multiple tokens found, but only one is allowed

GoReleaser infers if you are using GitHub, GitLab or Gitea by which tokens are provided.
If you have multiple tokens set, you'll get this error.

Here's an example:

```sh
   ⨯ release failed after 0.02s error=gmultiple tokens found, but only one is allowed: GITHUB_TOKEN, GITLAB_TOKEN

Learn more at https://goreleaser.com/errors/multiple-tokens
```

In this case, you either unset `GITHUB_TOKEN` or `GITLAB_TOKEN`.
You can read more about it in the [SCM docs](/scm/github/).
