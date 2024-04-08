---
date: 2022-02-05
slug: cloud-native-storage
categories:
  - tutorials
authors:
  - dirien
---

# How to use GoReleaser with Cloud Native Storage

In this tutorial, I want to describe, how quickly we can deploy our release
artifacts to a cloud native storage when using GoReleaser.
It’s just a few additional lines in your `.goreleaser.yaml`.

<!-- more -->

To better show this, I created a little demo and use the storage services of the
big three cloud providers: Azure Blob Storage, AWS S3 and Google Cloud Storage.

![](https://cdn-images-1.medium.com/max/2000/1*4kvgGvBM9--v2rS7nO5c1g.png)

You can use any S3 compatible storage provider too.
**GoReleaser** support this too! The most prominent (self-hosted) solution is
**MinIO**.

![](https://cdn-images-1.medium.com/max/4802/1*SH5PQKBDEB0M8mAY7EONeQ.png)

## The infrastructure code

I created a very simple **Terraform** deployment to provision on all three cloud
provider their appropriate cloud storage service.
It’s a demo, why not?

You don’t need to use **Terraform** for this, you could use any other means like
**Pulumi**, **CLI** or even the **UI**.

###### `main.tf`

```terraform
terraform {
  required_providers {
    google  = {
      source  = "hashicorp/google"
      version = "4.9.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "2.94.0"
    }
    aws     = {
      source  = "hashicorp/aws"
      version = "3.74.0"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "google" {
  credentials = file(var.gcp_auth_file)
  project     = var.gcp_project
  region      = var.gcp_region
}

provider "aws" {
  region = var.aws_region
}
```

###### `variables.tf`

```terraform
variable "gcp_project" {
  type = string
}

variable "gcp_region" {
  default = "europe-west6"
}

variable "gcp_zone" {
  default = "europe-west6-a"
}

variable "gcp_bucket_location" {
  default = "EU"
}

variable "gcp_auth_file" {
  default = "./auth.json"
  description = "Path to the GCP auth file"
}

variable "aws_region" {
  default = "eu-central-1"
}

variable "azure_location" {
  default = "West Europe"
}

variable "name" {
  default = "goreleaser-quickbites"
}
```

###### `blob.tf`

```terraform
resource "google_storage_bucket" "goreleaser-gcp-storage-bucket" {
  name                        = var.name
  location                    = var.gcp_bucket_location
  force_destroy               = true
  uniform_bucket_level_access = false
}
resource "google_storage_bucket_access_control" "goreleaser-gcp-storage-bucket-access-control" {
  bucket = google_storage_bucket.goreleaser-gcp-storage-bucket.name
  role   = "READER"
  entity = "allUsers"
}

resource "azurerm_resource_group" "goreleaser-azure-resource-group" {
  name     = var.name
  location = var.azure_location
}

resource "azurerm_storage_account" "goreleaser-azure-storage-account" {
  name                     = "gorleaserquickbites"
  resource_group_name      = azurerm_resource_group.goreleaser-azure-resource-group.name
  location                 = azurerm_resource_group.goreleaser-azure-resource-group.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  allow_blob_public_access = true
  network_rules {
    default_action = "Allow"
  }
}

resource "azurerm_storage_container" "goreleaser-storage-container" {
  name                  = var.name
  storage_account_name  = azurerm_storage_account.goreleaser-azure-storage-account.name
  container_access_type = "container"
}

resource "aws_s3_bucket" "goreleaser-s3-bucket" {
  bucket = var.name
  acl    = "public-read"
}
```

###### Apply the Terraform script:

```bash
terraform apply  -var  "gcp_project=xxx"
```

```
...
azurerm_storage_container.goreleaser-storage-container: Creation complete after 0s [id=https://goreleaserquickbites.blob.core.windows.net/goreleaser-quickbites]

Apply complete! Resources: 6 added, 0 changed, 0 destroyed.

Outputs:

aws-s3-bucket-name = "goreleaser-quickbites"
azure-storage-account-key = <sensitive>
azure-storage-account-name = "export AZURE_STORAGE_ACCOUNT=goreleaserquickbites"
gcp-bucket-url = "gs://goreleaser-quickbites"
```

###### Run this command

```bash
terraform output azure-storage-account-key
```

to get the Azure Storage Account Key, as it is a output field with sensitive data in it.

```bash
export AZURE_STORAGE_KEY=xxxx
```

Now we can add in our `.goreleaser.yaml` the new **blobs** field.
Important is here to set the right provider: **gs** (for Google Cloud Storage),
**azblob** (for Azure Blob) and **s3** (for AWS S3 or compatible provider)!

```yaml
# This is an example .goreleaser.yml file with some sensible defaults.
# Make sure to check the documentation at https://goreleaser.com
before:
  hooks:
    - go mod tidy
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin

release:
  disable: true
---
blobs:
  - provider: gs
    bucket: goreleaser-quickbites
  - provider: azblob
    bucket: goreleaser-quickbites
  - provider: s3
    bucket: goreleaser-quickbites
    region: eu-central-1
```

In this demo, I disabled the **release **section, as I don’t want to upload to
GitHub.

## Authentication

In terms of authentication the GoReleaser’s blob pipe authentication varies depending upon the blob provider as mentioned below:

### S3 Provider

S3 provider support AWS [default credential
provider](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials)
chain in the following order:

- Environment variables.
- Shared credentials file.
- If your application is running on an Amazon EC2 instance, IAM role for Amazon EC2.

### Azure Blob Provider Currently it supports authentication only

with [environment variables](https://docs.microsoft.com/en-us/azure/storage/common/storage-azure-cli#set-default-azure-storage-account-environment-variables):

- AZURE_STORAGE_ACCOUNT
- AZURE_STORAGE_KEY or AZURE_STORAGE_SAS_TOKEN

### GCS

Provider GCS provider uses [Application Default
Credentials](https://cloud.google.com/docs/authentication/production) in the
following order:

- Environment Variable (GOOGLE_APPLICATION_CREDENTIALS)
- Default Service Account from the compute instance (Compute Engine, Kubernetes
  Engine, Cloud function etc).

## Run GoReleaser

After configuring we can finally execute **GoReleaser**, in your pipeline code
via the command:

```bash
goreleaser release --rm-dist
```

If everything went smooth, you should see a similar output, showing the upload of your artifacts.

```
  ...
   • publishing
   • blobs
   • uploading path=quick-bites/0.1/quick-bites_0.1_checksums.txt
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_amd64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_amd64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_386.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_checksums.txt
   • uploading path=quick-bites/0.1/quick-bites_0.1_checksums.txt
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_386.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_amd64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_amd64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_amd64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_386.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_linux_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_arm64.tar.gz
   • uploading path=quick-bites/0.1/quick-bites_0.1_darwin_amd64.tar.gz
   • release succeeded after 22.63s
  ...
```

> One note: The provider fails silently, if your credentials are wrong. You
> would still see uploading and release succeeded. Keep this in mind, if the
> files are not appearing in the UI. I wasted some time on this. The culprit is
> the underlying library GoReleaser is using.

Let’s check in the consoles of the cloud provider too, If the files are present.

###### Google Cloud Storage:

![Google Cloud Storage](https://cdn-images-1.medium.com/max/2468/1*OHPaMIOK2YP7HsdgXEPpSw.png)

###### Azure Blob Storage

![Azure Blob Storage](https://cdn-images-1.medium.com/max/2792/1*K0BMoKe2qH29YHCOtZ8e9A.png)

###### AWS S3

![AWS S3](https://cdn-images-1.medium.com/max/2868/1*mvgyZMWtZseRabusw_R4Dg.png)

Looks very good! Now you can share the URLs of the files for further use!

## Want more Informations?

If you want to know more about some advanced options, feel free to check out the
[official documentation about the blob support in
GoReleaser](https://goreleaser.com/customization/blob/)

And here is the example code: [dirien/quick-bytes](https://github.com/dirien/quick-bites/tree/main/goreleaser-blob)

![Have fun](https://cdn-images-1.medium.com/max/2000/0*kbicxfar7Vo9rUon.jpg)
