# Changelog

You can customize how the changelog is generated using the `changelog` section in the config file:

```yaml
# .goreleaser.yml
changelog:
  # Set this to true if you don't want any changelog at all.
  #
  # Warning: this will also ignore any changelog files passed via `--release-notes`,
  # and will render an empty changelog.
  #
  # This may result in an empty release notes on GitHub/GitLab/Gitea.
  #
  # Templateable since v1.16.0.
  # Must evaluate to either true or false.
  skip: '{{ .Env.CREATE_CHANGELOG }}'

  # Changelog generation implementation to use.
  #
  # Valid options are:
  # - `git`: uses `git log`;
  # - `github`: uses the compare GitHub API, appending the author login to the changelog.
  # - `gitlab`: uses the compare GitLab API, appending the author name and email to the changelog.
  # - `github-native`: uses the GitHub release notes generation API, disables the groups feature.
  #
  # Default: 'git'
  use: github

  # Sorts the changelog by the commit's messages.
  # Could either be asc, desc or empty
  sort: asc

  # Max commit hash length to use in the changelog.
  #
  # 0: use whatever the changelog implementation gives you
  # -1: remove the commit hash from the changelog
  # any other number: max length.
  #
  # Since: v1.11.2
  abbrev: -1

  # Paths to filter the commits for.
  # Only works when `use: git`, otherwise ignored.
  # Only on GoReleaser Pro.
  #
  # Default: monorepo.dir value, or empty if no monorepo
  # Since: v1.12 (pro)
  # This feature is only available in GoReleaser Pro.
  paths:
  - foo/
  - bar/

  # Group commits messages by given regex and title.
  # Order value defines the order of the groups.
  # Providing no regex means all commits will be grouped under the default group.
  # Groups are disabled when using github-native, as it already groups things by itself.
  # Matches are performed against strings of the form: "<abbrev-commit>[:] <title-commit>".
  # Regex use RE2 syntax as defined here: https://github.com/google/re2/wiki/Syntax.
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: 'Bug fixes'
      regexp: '^.*?bug(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999
      # A group can have subgroups.
      # If you use this, all the commits that match the parent group will also
      # be checked against its subgroups. If some of them matches, it'll be
      # grouped there, otherwise they'll remain ungrouped.
      #
      # The title is optional - you can think of groups as a way to order
      # commits within a group.
      #
      # There can only be one level of subgroups, i.e. a subgroup cannot have
      # subgroups.
      #
      # This feature is only available in GoReleaser Pro.
      #
      # Since: v1.15 (pro)
      # This feature is only available in GoReleaser Pro.
      subgroups:
        - title: 'Docs'
          regex: '.*docs.*'
          order: 1
        - title: 'CI'
          regex: '.*build.*'
          order: 2

  # Divider to use between groups.
  #
  # Default: ''
  # Since: v1.16 (pro)
  # This feature is only available in GoReleaser Pro.
  divider: '---'

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
    Some things to keep an eye on:

    * The `github-native` changelog does not support `sort` and `filter`.
    * When releasing a [nightly][], `use` will fallback to `git`.

[nightly]: /customization/nightly
