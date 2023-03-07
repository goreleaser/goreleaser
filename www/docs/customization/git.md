# Git

> Since v1.14.0.

This allows you to change the behavior of some Git commands.

```yaml
# .goreleaser.yaml
git:
  # What should be used to sort tags when gathering the current and previous
  # tags if there are more than one tag in the same commit.
  #
  # Default: `-version:refname`
  tag_sort: -version:creatordate

  # What should be used to specify prerelease suffix while sorting tags when gathering
  # the current and previous tags if there are more than one tag in the same commit.
  #
  # Default: `-`
  prerelease_suffix: +
```
