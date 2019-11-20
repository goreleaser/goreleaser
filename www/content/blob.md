---
title: Blob
series: customization
hideFromIndex: true
weight: 114
---

## Customization

```yaml
# .goreleaser.yml
blobs:
  # You can have multiple blob configs
  -
    # Template for the cloud provider name
    # s3 for AWS S3 Storage
    # azblob for Azure Blob Storage
    # gs for Google Cloud Storage
    provider: azblob

    # Set a custom endpoint, useful if you're using a minio backend or
    # other s3-compatible backends.
    # Implies s3ForcePathStyle and requires provider to be `s3`
    endpoint: https://minio.foo.bar

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

Goreleaser's blob pipe authentication varies depending upon the blob provider as mentioned below:

### S3 Provider

S3 provider support AWS [default credential provider](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials) chain in the following order:

- Environment variables.

- Shared credentials file.

- If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

### Azure Blob Provider

Currently it supports authentication only with [environment variables](https://docs.microsoft.com/en-us/azure/storage/common/storage-azure-cli#set-default-azure-storage-account-environment-variables):

- AZURE_STORAGE_ACCOUNT
- AZURE_STORAGE_KEY or AZURE_STORAGE_SAS_TOKEN

### [GCS Provider](https://cloud.google.com/docs/authentication/production)

GCS provider uses [Application Default Credentials](https://cloud.google.com/docs/authentication/production) in the following order:

- Environment Variable (GOOGLE_APPLICATION_CREDENTIALS)
- Default Service Account from the compute instance(Compute Engine, Kubernetes Engine, Cloud function etc).

