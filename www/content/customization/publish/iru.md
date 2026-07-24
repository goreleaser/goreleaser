---
title: "Iru Custom Apps"
linkTitle: Iru
weight: 165
---

{{< g_version "v2.18" >}}

GoReleaser can publish artifacts as _Custom Apps_ to
[iru.com](https://www.iru.com) (formerly Kandji) endpoint management, making
them available for deployment to your managed Macs.

## How it works

Publishing follows the three-step flow of the
[Iru API](https://api-docs.iru.com): GoReleaser requests pre-signed S3 upload
details, uploads the file to S3, and then creates (or updates) the Custom App
library item.

Custom Apps are a macOS-only library item: the supported install types
(`package`, `zip`, `image`) correspond to `.pkg`, `.zip`, and `.dmg` files,
and the pre/post install scripts run on the target Mac.

Prerequisites:

- An [Iru API token](https://support.kandji.io/api), read from the
  `$IRU_API_TOKEN` environment variable by default. Permissions are granted
  per endpoint, so the token needs the following Library permissions:
  - `Upload Custom App`: always required.
  - `Create Custom App`: required when creating a new Custom App on each
    release (no `library_item_id` set).
  - `Update Custom App`: required when `library_item_id` is set.

The `iru` section specifies how the Custom App should be created:

```yaml {filename=".goreleaser.yaml"}
iru:
  # Your Iru API base URL.
  # You can find it in Settings > Access, e.g.
  # US: https://SubDomain.api.kandji.io
  # EU: https://SubDomain.api.eu.kandji.io
  #
  # Required.
  # Templates: allowed.
  url: https://mycompany.api.kandji.io

  # Name of the Custom App in the Library.
  #
  # If more than one artifact matches, each one is published as its own
  # Custom App, so make sure the name is unique per artifact, e.g. by using
  # templates like {{ .Os }} or {{ .Arch }}.
  #
  # Default: the project name.
  # Templates: allowed (artifact fields available).
  name: "My App {{ .Version }}"

  # IDs of the artifacts to publish.
  #
  # Default: all uploadable archives, binaries, and files.
  ids:
    - macos-pkg

  # API token.
  #
  # Default: the $IRU_API_TOKEN environment variable.
  # Templates: allowed.
  api_token: "{{ .Env.MY_IRU_TOKEN }}"

  # ID of an existing Custom App library item to update instead of creating
  # a new one on every release.
  #
  # When set, exactly one artifact must match the given ids.
  #
  # Templates: allowed.
  library_item_id: 58429143-b55c-42d3-a9a3-7c699ddd0ce1

  # How the file should be installed.
  #
  # Valid options: package, zip, image.
  # Default: package.
  install_type: package

  # Installation enforcement.
  #
  # Valid options: install_once, continuously_enforce, no_enforcement.
  # Default: install_once.
  install_enforcement: install_once

  # Path to extract a zip file to.
  #
  # Required if install_type is zip.
  unzip_location: /Applications

  # Audit script.
  #
  # Required if install_enforcement is continuously_enforce.
  audit_script: ""

  # Script to run before the install.
  preinstall_script: ""

  # Script to run after the install.
  postinstall_script: ""

  # Whether to show the app in Self Service.
  #
  # If not set, the field is not sent, so updates keep whatever is
  # configured in the Iru dashboard.
  show_in_self_service: false

  # Self Service category ID to display the app in.
  #
  # Required if show_in_self_service is true.
  # Templates: allowed.
  self_service_category_id: ""

  # Whether to flag the app as recommended in Self Service.
  #
  # If not set, the field is not sent, so updates keep whatever is
  # configured in the Iru dashboard.
  self_service_recommended: false

  # Whether to restart the device after a successful install.
  #
  # If not set, the field is not sent, so updates keep whatever is
  # configured in the Iru dashboard.
  restart: false

  # Whether to disable this feature.
  #
  # Templates: allowed.
  disable: "{{ .IsSnapshot }}"
```

{{< g_templates >}}
