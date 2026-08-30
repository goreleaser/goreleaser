---
title: "Report Sizes"
weight: 36
---

Enable this to keep an eye on your binary and package sizes.

It'll report the size of each artifact of the following types to the build
output, as well as on `dist/artifacts.json`:

- `Binary`
- `UniversalBinary`
- `UploadableArchive`
- `UploadableBinary`
- `UploadableFile`
- `UploadableSourceArchive`
- `PublishableSnapcraft`
- `LinuxPackage`
- `Makeself`
- `MSIX`
- `Flatpak`
- `SourceRPM`
- `SBOM`
- `PyWheel`
- `PySdist`
- `Checksum`
- `Signature`
- `Certificate`
- `CArchive`
- `CShared`
- `Header`

Here are the available configuration options:

```yaml {filename=".goreleaser.yaml"}
# Whether to enable the size reporting or not.
report_sizes: true
```
