# Changelog

You can customize how the changelog is generated using the `changelog` section in the config file:

```yaml
# .goreleaser.yml
changelog:
  # Set it to true if you wish to skip the changelog generation.
  # This may result in an empty release notes on GitHub/GitLab/Gitea.
  skip: true

  # Implementation to use to generate the changelog.
  # Valid options are `git` and `github`.
  # Defaults to `git`.
  impl: github

  # Sorts the changelog by the commit's messages.
  # Could either be asc, desc or empty
  # Default is empty
  sort: asc

  filters:
    # Commit messages matching the regexp listed here will be removed from
    # the changelog
    # Default is empty
    exclude:
      - '^docs:'
      - typo
      - (?i)foo
```
