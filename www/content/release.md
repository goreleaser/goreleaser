---
title: Release
series: customization
hideFromIndex: true
weight: 110
---

GoReleaser will create a GitHub release with the current tag, upload all
the artifacts and generate the changelog based on the new commits since the
previous tag.

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
  # Default is ``
  name_template: "{{.ProjectName}}-v{{.Version}} {{.Env.USER}}"

  # You can disable this pipe in order to not upload any artifacts to
  # GitHub.
  # Defaults to false.
  disable: true
```

> Learn more about the [name template engine](/templates).

## Customize the changelog

You can customize how the changelog is generated using the
`changelog` section in the config file:

```yaml
# .goreleaser.yml
changelog:
  # could either be asc, desc or empty
  # Default is empty
  sort: asc
  filters:
    # commit messages matching the regexp listed here will be removed from
    # the changelog
    # Default is empty
    exclude:
      - '^docs:'
      - typo
      - (?i)foo
```

## Custom release notes

You can specify a file containing your custom release notes, and
pass it with the `--release-notes=FILE` flag.
GoReleaser will then skip its own release notes generation,
using the contents of your file instead.
You can use Markdown to format the contents of your file.

On Unix systems you can also generate the release notes in-line by using
[process substitution](https://en.wikipedia.org/wiki/Process_substitution).
To list all commits since the last tag, but skip ones starting with `Merge` or
`docs`, you could run this command:

```console
$ goreleaser --release-notes <(some_changelog_generator)
```

Some changelog generators you can use:

- [buchanae/github-release-notes](https://github.com/buchanae/github-release-notes)
