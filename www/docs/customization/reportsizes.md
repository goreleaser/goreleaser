# Report Sizes

> Since v1.18

You might want to enable this if you want to keep an eye on your binary/package
sizes.

It'll report the size of each artifact of the following types to the build
output, as well as on `dist/artifacts.json`:

- `Binary,`
- `UniversalBinary,`
- `UploadableArchive,`
- `PublishableSnapcraft,`
- `LinuxPackage,`
- `CArchive,`
- `CShared,`
- `Header,`

Here's the available configuration options:

```yaml
# .goreleaser.yaml
# Whether to enable the size reporting or not.
report_sizes: true
```
