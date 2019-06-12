---
title: Blob
series: customization
---

## Customization

```yaml
# .goreleaser.yml
blob:
  # You can have multiple blob configs
  -
    # Template for the cloud provider name
    # s3 for AWS S3 Storage
    # azblob for Azure Blob Storage
    # gs for Google Cloud Storage
    provider: azblob

    # Template for the bucket name
    bucket: goreleaser-bucket

    # IDs of the artifacts you want to upload.
    ids:
    - foo
    - bar

    # Template for the path/name inside the bucket.
    # Default is `{{ .ProjectName }}/{{ .Tag }}`
    folder: "foo/bar/{{.Version}}"
  -
    provider: gs
    bucket: goreleaser-bucket
    folder: "foo/bar/{{.Version}}"
  -
    provider: s3
    bucket: goreleaser-bucket
    folder: "foo/bar/{{.Version}}"
```

> Learn more about the [name template engine](/templates).

## Authentication

Currently it supports authentication only with Environment Variable, Below is the list of ENV variable required:

### [S3 Provider](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-envvars.html)

- AWS_ACCESS_KEY
- AWS_SECRET_KEY
- AWS_DEFAULT_REGION

### [Azure Blob Provider](https://docs.microsoft.com/en-us/azure/storage/common/storage-azure-cli#set-default-azure-storage-account-environment-variables)

- AZURE_STORAGE_ACCOUNT
- AZURE_STORAGE_KEY

### [GCS Provider](https://cloud.google.com/docs/authentication/production)

- GOOGLE_APPLICATION_CREDENTIALS
