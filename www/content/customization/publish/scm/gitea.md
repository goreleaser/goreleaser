---
title: "Gitea"
weight: 30
---

## Uploading `tar.gz` and `checksums.txt`

To enable uploading `tar.gz` and `checksums.txt` files you need to add the
following to your Gitea config in `app.ini`:

```ini
[attachment]
ALLOWED_TYPES = application/gzip|application/x-gzip|application/x-gtar|application/x-tgz|application/x-compressed-tar|text/plain
```

> [!WARNING]
> Gitea versions earlier than 1.9.2 do not support uploading `checksums.txt`
> files because of a [bug](https://github.com/go-gitea/gitea/issues/7882), so
> you will have to enable all file types with `*/*`.

## API Token

GoReleaser requires an API token to deploy the artifacts to Gitea.
You can create one in `Settings | Applications | Generate New Token` page of your Gitea instance.

This token should be added to the environment variables as `GITEA_TOKEN`.

Alternatively, you can provide the Gitea token in a file.
GoReleaser will check `~/.config/goreleaser/gitea_token` by default, but you can change that in the `.goreleaser.yaml` file:

```yaml {filename=".goreleaser.yaml"}
env_files:
  gitea_token: ~/.path/to/my/gitea_token
```

Note that the environment variable will be used if available, regardless of the
`gitea_token` file.

## URLs

You can use GoReleaser with Gitea by providing its URLs in
the `.goreleaser.yaml` configuration file. This takes a normal string, or a template value.

```yaml {filename=".goreleaser.yaml"}
gitea_urls:
  api: https://gitea.myinstance.com/api/v1
  download: https://gitea.myinstance.com
  # set to true if you use a self-signed certificate
  skip_tls_verify: false
```
