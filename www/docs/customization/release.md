---
title: Release
---

GoReleaser will create a GitHub/GitLab release with the current tag, upload all
the artifacts and generate the changelog based on the new commits since the
previous tag.

Let's see what can be customized in the `release` section for GitHub:

```yaml
# .goreleaser.yml
release:
  # Repo in which the release will be created.
  # Default is extracted from the origin remote URL or empty if its private hosted.
  # Note: it can only be one: either github or gitlab or gitea
  github:
    owner: user
    name: repo

  # IDs of the archives to use.
  # Defaults to all.
  ids:
    - foo
    - bar

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

  # You can add extra pre-existing files to the release.
  # The filename on the release will be the last part of the path (base). If
  # another file with the same name exists, the latest one found will be used.
  # Defaults to empty.
  extra_files:
    - glob: ./path/to/file.txt
    - glob: ./glob/**/to/**/file/**/*
    - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
```

Second, let's see what can be customized in the `release` section for GitLab.

```yaml
# .goreleaser.yml
release:
  # Same as for github
  # Note: it can only be one: either github or gitlab or gitea
  gitlab:
    owner: user
    name: repo

  # IDs of the archives to use.
  # Defaults to all.
  ids:
    - foo
    - bar

  # You can change the name of the GitLab release.
  # Default is `{{.Tag}}`
  name_template: "{{.ProjectName}}-v{{.Version}} {{.Env.USER}}"

  # You can disable this pipe in order to not upload any artifacts to
  # GitLab.
  # Defaults to false.
  disable: true

  # You can add extra pre-existing files to the release.
  # The filename on the release will be the last part of the path (base). If
  # another file with the same name exists, the latest one found will be used.
  # Defaults to empty.
  extra_files:
    - glob: ./path/to/file.txt
    - glob: ./glob/**/to/**/file/**/*
    - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
```

!!! warning
    Only GitLab `v11.7+` are supported for releases.

You can also configure the `release` section to upload to a [Gitea](https://gitea.io) instance:

```yaml
# .goreleaser.yml
release:
  # Same as for github and gitlab
  # Note: it can only be one: either github or gitlab or gitea
  gitea:
    owner: user
    name: repo

  # IDs of the artifacts to use.
  # Defaults to all.
  ids:
    - foo
    - bar

  # You can change the name of the Gitea release.
  # Default is `{{.Tag}}`
  name_template: "{{.ProjectName}}-v{{.Version}} {{.Env.USER}}"

  # You can disable this pipe in order to not upload any artifacts to
  # Gitea.
  # Defaults to false.
  disable: true

  # You can add extra pre-existing files to the release.
  # The filename on the release will be the last part of the path (base). If
  # another file with the same name exists, the latest one found will be used.
  # Defaults to empty.
  extra_files:
    - glob: ./path/to/file.txt
    - glob: ./glob/**/to/**/file/**/*
    - glob: ./glob/foo/to/bar/file/foobar/override_from_previous
```

To enable uploading `tar.gz` and `checksums.txt` files you need to add the following to your Gitea config in `app.ini`:
```ini
[attachment]
ALLOWED_TYPES = application/gzip|application/x-gzip|application/x-gtar|application/x-tgz|application/x-compressed-tar|text/plain
```

!!! warning
    Gitea versions earlier than 1.9.2 do not support uploading `checksums.txt`
    files because of a [bug](https://github.com/go-gitea/gitea/issues/7882)
    so you will have to enable all file types with `*/*`.

!!! warning
    `draft` and `prerelease` are only supported by GitHub and Gitea.

!!! tip
    Learn more about the [name template engine](/customization/templates).

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

### Define Previous Tag

GoReleaser uses `git describe` to get the previous tag used for generating the Changelog.
You can set a different build tag using the environment variable `GORELEASER_PREVIOUS_TAG`.
This is useful in scenarios where two tags point to the same commit.

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

!!! info
    If you create the release before running GoReleaser, and the
    said release has some text in its body, GoReleaser will not override it with
    it's release notes.
