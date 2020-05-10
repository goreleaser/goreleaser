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
  # Default is `{{ .ProjectName }}_{{ .Version }}_checksums.txt`.
  name_template: "{{ .ProjectName }}_checksums.txt"

  # Algorithm to be used.
  # Accepted options are sha256, sha512, sha1, crc32, md5, sha224 and sha384.
  # Default is sha256.
  algorithm: sha256
```

> Learn more about the [name template engine](/templates).
