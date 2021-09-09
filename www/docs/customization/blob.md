---
title: Blobs
---

The `blobs` allows you to upload artifacts to Amazon S3, Azure Blob and
Google GCS.

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

    # Sets the bucket region.
    # Requires provider to be `s3`
    # Defaults to empty.
    region: us-west-1

    # Disables SSL
    # Requires provider to be `s3`
    # Defaults to false
    disableSSL: true

    # Template for the bucket name
    bucket: goreleaser-bucket

    # IDs of the artifacts you want to upload.
    ids:
    - foo
    - bar

    # Template for the path/name inside the bucket.
    # Default is `{{ .ProjectName }}/{{ .Tag }}`
    folder: "foo/bar/{{.Version}}"

    # You can add extra pre-existing files to the release.
    # The filename on the release will be the last part of the path (base). If
    # another file with the same name exists, the last one found will be used.
    # Defaults to empty.
    extra_files:
      - glob: ./path/to/file.txt
      - glob: ./glob/**/to/**/file/**/*
      - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
  -
    provider: gs
    bucket: goreleaser-bucket
    folder: "foo/bar/{{.Version}}"
  -
    provider: s3
    bucket: goreleaser-bucket
    folder: "foo/bar/{{.Version}}"
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## Authentication

GoReleaser's blob pipe authentication varies depending upon the blob provider as mentioned below:

### S3 Provider

S3 provider support AWS
[default credential provider](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)
chain in the following order:

- Environment variables.
- Shared credentials file.
- If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

### Azure Blob Provider

Currently it supports authentication only with
[environment variables](https://docs.microsoft.com/en-us/azure/storage/common/storage-azure-cli#set-default-azure-storage-account-environment-variables):

- `AZURE_STORAGE_ACCOUNT`
- `AZURE_STORAGE_KEY` or `AZURE_STORAGE_SAS_TOKEN`

### [GCS Provider](https://cloud.google.com/docs/authentication/production)

GCS provider uses
[Application Default Credentials](https://cloud.google.com/docs/authentication/production)
in the following order:

- Environment Variable (`GOOGLE_APPLICATION_CREDENTIALS`)
- Default Service Account from the compute instance (Compute Engine,
Kubernetes Engine, Cloud function etc).

## ACLs

There is no common way to set ACLs across all bucket providers, so, [go-cloud][]
[does not support it yet][issue1108].

You are expected to set the ACLs on the bucket/folder/etc, depending on your
provider.

[go-cloud]: https://gocloud.dev/howto/blob/
[issue1108]: https://github.com/google/go-cloud/issues/1108
