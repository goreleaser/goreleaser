---
title: Custom checksum
---

GoRelease generates a `project_1.0.0_checksums.txt` and uploads it to the
release as well, so your users can validate if the downloaded files are
right.

The `checksum` section allows the customization of the filename:

```yml
# .goreleaser.yml
checksum:
  # You can change the name of the checksums file.
  # This is parsed with Golang template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Tag with the `v` prefix stripped)
  # The default is `{{ .ProjectName }}_{{ .Version }}_checksums.txt`
  name_template: "{{ .ProjectName }}_checksums.txt"
```
