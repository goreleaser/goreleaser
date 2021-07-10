# Gitea

## API Token

GoReleaser requires an API token to deploy the artifacts to Gitea.
You can create one in `Settings | Applications | Generate New Token` page of your Gitea instance.

This token should be added to the environment variables as `GITEA_TOKEN`.

Alternatively, you can provide the Gitea token in a file.
GoReleaser will check `~/.config/goreleaser/gitea_token` by default, but you can change that in the `.goreleaser.yml` file:

```yaml
# .goreleaser.yml
env_files:
  gitea_token: ~/.path/to/my/gitea_token
```

## URLs

You can use GoReleaser with Gitea by providing its URLs in
the `.goreleaser.yml` configuration file:

```yaml
# .goreleaser.yml
gitea_urls:
  api: https://gitea.myinstance.com/api/v1/
  download: https://gitea.myinstance.com
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```
