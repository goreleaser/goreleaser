---
title: Releasing
---

GoReleaser will create a release in GitHub with the current tag, upload all
the archives and checksums, also generate a changelog from the commits new since the last tag.

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
  # Default is false.
  draft: true

  # If set to true, will mark the release as not ready for production.
  # Default is false.
  prerelease: true

  # You can change the name of the GitHub release.
  # This is parsed with the Go template engine and the following variables
  # are available:
  # - ProjectName
  # - Tag
  # - Version (Git tag without `v` prefix)
  # Default is ``
  name_template: "{{.ProjectName}}-v{{.Version}}"
```

## Custom release notes

You can specify a file containing your custom release notes, and
pass it with the `--release-notes=FILE` flag.
GoReleaser will then skip its own release notes generation,
using the contents of your file instead.
You can use Markdown to format the contents of your file.

On Unix systems you can also generate the release notes in-line by using [process substitution](https://en.wikipedia.org/wiki/Process_substitution).
To list all commits since the last tag, but skip ones starting with `Merge` or `docs`, you could run this command:

```sh
goreleaser --release-notes <(git log --pretty=oneline --abbrev-commit $(git describe --tags --abbrev=0)^.. | grep -v '^[^ ]* \(Merge\|docs\)')
```
