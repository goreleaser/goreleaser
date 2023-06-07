# Archive has different count of binaries for each platform

This error looks like this:

```sh
⨯ release failed after 5s                  error=invalid archive: 0:archive has different count of binaries for each platform, which may cause your users confusion.
Learn more at https://goreleaser.com/errors/multiple-binaries-archive

```

This will happen when you have several builds, and their target platforms are
different:

```yaml
builds:
  - id: b1
    binary: b1
    goos: [linux, darwin]
  - id: b2
    binary: b2
    goos: [darwin]

archives:
  - id: a1
```

In this scenario, GoReleaser will complain because the archive will have a
different binary count depending on which platform its being archived, since
it'll have 2 binaries on `darwin` and only 1 on `linux`.

From here on, you have a couple of options:

- add another archive, and filter the builds on each of them - e.g. archive `a1`
  with binaries from build `b1`, and archive `a2` with builds from build `b2`:
  ```yaml
  archives:
    - id: a1
      builds: [b1]
      name_template: something-unique-for-a1
    - id: a2
      builds: [b2]
      name_template: something-unique-for-a2
  ```
- if you really want to have the mixed archive, you can add
  `allow_different_binary_count` to your archive configuration:
  ```yaml
  archives:
    - id: a1
      allow_different_binary_count: true
  ```
