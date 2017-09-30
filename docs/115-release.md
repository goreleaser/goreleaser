---
title: Custom releasing
---

GoRelease will create a release in GitHub with the current tag, upload all
the archives and checksums, also generating a changelog from the commit
log between the current and previous tags.

Let's see what can be customized in the `release` section:

```yml
# .goreleaser.yml
release:
  # Repo in which the release will be created.
  # Default is extracted from the origin remote URL.
  github:
    owner: user
    name: repo

  # If set to true, will not auto-publish the release.
  # Default is false
  draft: true
```

## Custom release notes

You can have a markdown file previously created with the release notes, and
pass it down to goreleaser with the `--release-notes=FILE` flag.
GoReleaser will then skip its own release notes generation,
using the contents of your file instead.

On Unix systems you can also generate the release notes in-line by using [process substitution](https://en.wikipedia.org/wiki/Process_substitution).
To list all commits since the last tag, but skip ones starting with `Merge` or `docs`, you could run this command:

```sh
goreleaser --release-notes <(git log --pretty=oneline --abbrev-commit $(git describe --tags --abbrev=0)^.. | grep -v '^[^ ]* \(Merge\|docs\)')
```
