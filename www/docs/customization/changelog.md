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
  # - `github-native`: uses the GitHub release notes generation API, disables groups, sort, and any further formatting features.
  #
  # Default: 'git'.
  use: github

  # Format to use for commit formatting.
  #
  # Templates: allowed.
  #
  # Default:
  #    if 'git': '{{ .SHA }} {{ .Message }}'
  #   otherwise: '{{ .SHA }}: {{ .Message }} ({{ with .Author.Username }}@{{ . }}{{ else }}{{ .Author.Name }} <{{ .Author.Email }}>{{ end }})'.
  #
  # Extra template fields:
  # - `SHA`: the commit SHA1
  # - `Message`: the first line of the commit message, otherwise known as commit subject
  # - `Authors`: all authors of the commit
  # - `Logins`: all non-empty logins of the authors of the commit, prefixed with an '@' (not available if 'git') (since v2.14)
  #
  # An `Author` is composed of:
  # - `Name`: the author full name (considers mailmap if 'git')
  # - `Email`: the author email (considers mailmap if 'git')
  # - `Username`: github/gitlab/gitea username - not available if 'git', might be empty
  #
  # Deprecated in v2.14:
  # - `AuthorName`: the author full name (considers mailmap if 'git')
  # - `AuthorEmail`: the author email (considers mailmap if 'git')
  # - `AuthorUsername`: github/gitlab/gitea username - not available if 'git'
  #
  # Usage with 'github': <!-- md:inline_version v2.8 -->.
  format: "{{.SHA}}: {{.Message}}{{ if .Logins }} ({{ .Logins | englishJoin }}){{ end }}"

  # Sorts the changelog by the commit's messages.
  # Could either be asc, desc or empty
  # Empty means 'no sorting', it'll use the output of `git log` as is.
  #
  # Disabled when using 'github-native'.
  sort: asc

  # Max commit hash length to use in the changelog.
  #
  # 0: use whatever the changelog implementation gives you
  # -1: remove the commit hash from the changelog
  # any other number: max length.
  #
  # Disabled when using 'github-native'.
  abbrev: -1

  # Paths to filter the commits for.
  # Only works when `use: git`, otherwise ignored.
  #
  # This feature is only available in GoReleaser Pro.
  # Default: monorepo.dir value, or empty if no monorepo.
  #
  # Disabled when using 'github-native'.
  paths:
    - foo/
    - bar/

  # Compose your release notes with AI.
  # See below for more details.
  ai:
    use: anthropic
    prompt: "The prompt..."

  # Title of the changelog.
  #
  # Default: "Changelog".
  # <!-- md:inline_pro -->.
  # <!-- md:inline_version v2.12 -->.
  # Templates: allowed.
  title: "Release Notes"

  # Group commits messages by given regex and title.
  # Order value defines the order of the groups.
  # Providing no regex means all commits will be grouped under the default group.
  #
  # Matches are performed against the first line of the commit message only,
  # prefixed with the commit SHA1, usually in the form of
  # `<abbrev-commit>[:] <title-commit>`.
  # Regex use RE2 syntax as defined here: https://github.com/google/re2/wiki/Syntax.
  #
  # Disabled when using 'github-native'.
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

  # Further filter changelog entries.
  #
  # Disabled when using 'github-native'.
  filters:
    # Commit messages matching the regexp listed here will be removed from
    # the changelog
    #
    # Matches are performed against the first line of the commit message only.
    exclude:
      - "^docs:"
      - typo
      - (?i)foo

    # Commit messages matching the regexp listed here will be the only ones
    # added to the changelog
    #
    # If include is not-empty, exclude will be ignored.
    #
    # Matches are performed against the first line of the commit message only.
    include:
      - "^feat:"
```

!!! warning

    Some things to keep an eye on:

    * The `github-native` changelog does not support `groups`, `sort`, and `filter`.
    * When releasing a [nightly][], `use` will fallback to `git`.
    * The `github` changelog will only work if both tags exist in GitHub.

[nightly]: ./nightlies.md

## Enhance with AI

<!-- md:pro -->

<!-- md:version v2.6 -->

You can also use AI to enhance your release notes:

```yaml title=".goreleaser.yaml"
changelog:
  ai:
    # Which provider to use.
    # Will disable the feature if empty.
    # Enabling AI disables changelog grouping.
    #
    # Valid options: 'anthropic', `openai', 'ollama`.
    use: openai

    # Which model to use.
    #
    # Defaults to the provider's default model.
    model: o1-mini

    # The prompt to use..
    #
    # It can be a string directly, or you can use `from_url` or `from_file` to
    # source it from somewhere else.
    #
    # Templates: allowed.
    # Extra template fields available:
    # - `.ReleaseNotes`: will contain the release notes, with groups and
    #     everything else you have set up so far.
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

The default prompt will ask it to write a short intro with outlining the most
exciting features, merge dependency bumps of the same dependency together, and
to not use emojis.

You can of course set anything you wish makes sense in the `prompt` field.
Don't forget to give it the current release notes as well, available as
`{{ .ReleaseNotes }}`.

This is the [default
prompt](https://gist.githubusercontent.com/caarlos0/419c8cb2bab28f7c53c7e228af3ab219/raw/70e3e7f0ba85b02a23692d150e3a0d1752c79d64/prompt.md)
in case you're interested.

You can test this by using the
[`goreleaser changelog` command](../cmd/goreleaser_changelog.md).
