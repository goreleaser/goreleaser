---
title: Checksum
---

GoReleaser generates a `project_1.0.0_checksums.txt` file and uploads it with the
release, so your users can validate if the downloaded files are correct.

The `checksum` section allows customizations of the filename:

```yml
# .goreleaser.yml
checksum:
  # You can change the name of the checksums file.
  # This is parsed with the Go template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Git tag without `v` prefix)
  # - Env (environment variables)
  # Default is `{{ .ProjectName }}_{{ .Version }}_checksums.txt`.
  name_template: "{{ .ProjectName }}_checksums.txt"
```
