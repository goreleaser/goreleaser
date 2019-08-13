---
title: Release
series: customization
hideFromIndex: true
weight: 110
---

GoReleaser will create a GitHub/GitLab release with the current tag, upload all
the artifacts and generate the changelog based on the new commits since the
previous tag.

Let's see what can be customized in the `release` section for GitHub:

```yml
# .goreleaser.yml
release:
  # Repo in which the release will be created.
  # Default is extracted from the origin remote URL.
  # Note: it can only be one: either github or gitlab
  github:
    owner: user
    name: repo

  # If set to true, will not auto-publish the release.
  # Default is false.
  draft: true

  # If set to auto, will mark the release as not ready for production
  # in case there is an indicator for this in the tag e.g. v1.0.0-rc1
  # If set to true, will mark the release as not ready for production.
  # Default is false.
  prerelease: auto

  # You can change the name of the GitHub release.
  # Default is `{{.Tag}}`
  name_template: "{{.ProjectName}}-v{{.Version}} {{.Env.USER}}"

  # You can disable this pipe in order to not upload any artifacts to
  # GitHub.
  # Defaults to false.
  disable: true
```

Second, let's see what can be customized in the `release` section for GitLab.
**Note** that only GitLab `v11.7+` are supported for releases:

```yml
# .goreleaser.yml
release:
  # Same as for github
  # Note: it can only be one: either github or gitlab
  gitlab:
    owner: user
    name: repo

  # You can change the name of the GitLab release.
  # Default is `{{.Tag}}`
  name_template: "{{.ProjectName}}-v{{.Version}} {{.Env.USER}}"

  # You can disable this pipe in order to not upload any artifacts to
  # GitLab.
  # Defaults to false.
  disable: true
```

**Note**: `draft` and `prerelease` is only supported by GitHub.

> Learn more about the [name template engine](/templates).

## Customize the changelog

You can customize how the changelog is generated using the
`changelog` section in the config file:

```yaml
# .goreleaser.yml
changelog:
  # set it to true if you wish to skip the changelog generation
  skip: true
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

```sh
$ goreleaser --release-notes <(some_changelog_generator)
```

Some changelog generators you can use:

- [buchanae/github-release-notes](https://github.com/buchanae/github-release-notes)

> **Important**: If you create the release before running GoReleaser, and the
> said release has some text in its body, GoReleaser will not override it with
> it's release notes.
