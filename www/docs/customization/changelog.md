# Changelog

You can customize how the changelog is generated using the `changelog` section in the config file:

```yaml title=".goreleaser.yaml"
changelog:
  # Set this to true if you don't want any changelog at all.
  #
  # Warning: this will also ignore any changelog files passed via `--release-notes`,
  # and will render an empty changelog.
  #
  # This may result in an empty release notes on GitHub/GitLab/Gitea.
  #
  # Templates: allowed.
  disable: "{{ .Env.CREATE_CHANGELOG }}"

  # Changelog generation implementation to use.
  #
  # Valid options are:
  # - `git`: uses `git log`;
  # - `github`: uses the compare GitHub API, appending the author username to the changelog.
  # - `gitlab`: uses the compare GitLab API, appending the author name and email to the changelog (requires a personal access token).
  # - `gitea`: uses the compare Gitea API, appending the author username to the changelog.
  # - `github-native`: uses the GitHub release notes generation API, disables the groups feature.
  #
  # Default: 'git'.
  use: github

  # Format to use for commit formatting.
  # Only available when use is one of `github`, `gitea`, or `gitlab`.
  #
  # Default: '{{ .SHA }}: {{ .Message }} ({{ with .AuthorUsername }}@{{ . }}{{ else }}{{ .AuthorName }} <{{ .AuthorEmail }}>{{ end }})'.
  # Extra template fields: `SHA`, `Message`, `AuthorName`, `AuthorEmail`, and
  # `AuthorUsername`.
  format: "{{.SHA}}: {{.Message}} (@{{.AuthorUsername}})"

  # Sorts the changelog by the commit's messages.
  # Could either be asc, desc or empty
  # Empty means 'no sorting', it'll use the output of `git log` as is.
  sort: asc

  # Max commit hash length to use in the changelog.
  #
  # 0: use whatever the changelog implementation gives you
  # -1: remove the commit hash from the changelog
  # any other number: max length.
  abbrev: -1

  # Paths to filter the commits for.
  # Only works when `use: git`, otherwise ignored.
  #
  # This feature is only available in GoReleaser Pro.
  # Default: monorepo.dir value, or empty if no monorepo.
  paths:
    - foo/
    - bar/

  # Compose your release notes with AI.
  # See below for more details.
  ai:
    use: anthropic
    prompt: "The prompt..."

  # Group commits messages by given regex and title.
  # Order value defines the order of the groups.
  # Providing no regex means all commits will be grouped under the default group.
  #
  # Matches are performed against the first line of the commit message only,
  # prefixed with the commit SHA1, usually in the form of
  # `<abbrev-commit>[:] <title-commit>`.
  # Groups are disabled when using github-native, as it already groups things by itself.
  # Regex use RE2 syntax as defined here: https://github.com/google/re2/wiki/Syntax.
  groups:
    - title: Features
      regexp: '^.*?feat(\([[:word:]]+\))??!?:.+$'
      order: 0
    - title: "Bug fixes"
      regexp: '^.*?bug(\([[:word:]]+\))??!?:.+$'
      order: 1
    - title: Others
      order: 999

      # A group can have subgroups.
      # If you use this, all the commits that match the parent group will also
      # be checked against its subgroups. If some of them matches, it'll be
      # grouped there, otherwise they'll remain not grouped.
      #
      # The title is optional - you can think of groups as a way to order
      # commits within a group.
      #
      # There can only be one level of subgroups, i.e.: a subgroup can't have
      # subgroups within it.
      #
      # This feature is only available in GoReleaser Pro.
      groups:
        - title: "Docs"
          regex: ".*docs.*"
          order: 1
        - title: "CI"
          regex: ".*build.*"
          order: 2

  # Divider to use between groups.
  #
  # This feature is only available in GoReleaser Pro.
  divider: "---"

  filters:
    # Commit messages matching the regexp listed here will be removed from
    # the changelog
    #
    # Matches are performed against the first line of the commit message only,
    # prefixed with the commit SHA1, usually in the form of
    # `<abbrev-commit>[:] <title-commit>`.
    exclude:
      - "^docs:"
      - typo
      - (?i)foo

    # Commit messages matching the regexp listed here will be the only ones
    # added to the changelog
    #
    # If include is not-empty, exclude will be ignored.
    #
    # Matches are performed against the first line of the commit message only,
    # prefixed with the commit SHA1, usually in the form of
    # `<abbrev-commit>[:] <title-commit>`.
    include:
      - "^feat:"
```

!!! warning

    Some things to keep an eye on:

    * The `github-native` changelog does not support `sort` and `filter`.
    * When releasing a [nightly][], `use` will fallback to `git`.
    * The `github` changelog will only work if both tags exist in GitHub.

[nightly]: ./nightlies.md

## Enhance with AI

<!-- md:pro -->

<!-- md:version v2.6-unreleased -->

<!-- md:alpha -->

You can also use AI to enhance your release notes:

```yaml title=".goreleaser.yaml"
changelog:
  ai:
    # Which provider to use.
    # Will disable the feature if empty.
    # Enabling AI disables changelog grouping.
    #
    # Valid options: 'anthropic', `openai', 'ollama`.
    use: anthropic

    # Which model to use.
    #
    # Defaults to the provider's default model.
    model: claude-xyz

    # The prompt to use..
    #
    # It can be a string directly, or you can use `from_url` or `from_file` to
    # source it from somewhere else.
    #
    # Templates: allowed.
    # Extra template fields available:
    # - `.Commits`: will contain all the commits for this release, one per line.
    prompt:
      # Loads from an URL.
      from_url:
        # Templates: allowed.
        url: https://foo.bar/README.md
        headers:
          x-api-token: "${MYCOMPANY_TOKEN}"

      # Loads from a local file.
      # Overrides `from_url`.
      from_file:
        # Templates: allowed.
        path: ./README.md
```

You can test this feature by using the
[`goreleaser changelog` command](../cmd/goreleaser_changelog.md).
