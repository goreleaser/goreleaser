# Changelog

You can customize how the changelog is generated using the `changelog` section in the config file:

```yaml
# .goreleaser.yml
changelog:
  # Set this to true if you don't want any changelog at all.
  # Warning: this will also ignore any changelog files passed via `--release-notes`,
  # and will render an empty changelog.
  # This may result in an empty release notes on GitHub/GitLab/Gitea.
  skip: true

  # Changelog generation implementation to use.
  #
  # Valid options are:
  # - `git`: uses `git log`;
  # - `github`: uses the compare GitHub API, appending the author login to the changelog.
  # - `gitlab`: uses the compare GitLab API, appending the author name and email to the changelog.
  # - `github-native`: uses the GitHub release notes generation API, disables the groups feature.
  #
  # Defaults to `git`.
  use: github

  # Sorts the changelog by the commit's messages.
  # Could either be asc, desc or empty
  # Default is empty
  sort: asc

  # Max commit hash length to use in the changelog.
  #
  # 0: use whatever the changelog implementation gives you
  # -1: remove the commit hash from the changelog
  # any other number: max length.
  #
  # Default: 0.
  # Since: v1.11.2
  abbrev: -1

  # Paths to filter the commits for.
  # Only works when `use: git`, otherwise ignored.
  # Only on GoReleaser Pro.
  #
  # Default: monorepo.dir value, or empty if no monorepo.
  # Since: v1.12.0-pro
  paths:
  - foo/
  - bar/

  # Group commits messages by given regex and title.
  # Order value defines the order of the groups.
  # Proving no regex means all commits will be grouped under the default group.
  # Groups are disabled when using github-native, as it already groups things by itself.
  # Matches are performed against strings of the form: "<abbrev-commit>[:] <title-commit>".
  # Regex use RE2 syntax as defined here: https://github.com/google/re2/wiki/Syntax.
  #
  # Default is no groups.
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: 'Bug fixes'
      regexp: '^.*?bug(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

  filters:
    # Commit messages matching the regexp listed here will be removed from
    # the changelog
    # Default is empty
    exclude:
      - '^docs:'
      - typo
      - (?i)foo
```

!!! warning
    Note that using the `github-native` changelog does not support `sort` and `filter`.
