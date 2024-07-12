# Git

This allows you to change the behavior of some Git commands.

```yaml
# .goreleaser.yaml
git:
  # What should be used to sort tags when gathering the current and previous
  # tags if there are more than one tag in the same commit.
  #
  # Default: '-version:refname'.
  tag_sort: -version:creatordate

  # What should be used to specify prerelease suffix while sorting tags when gathering
  # the current and previous tags if there are more than one tag in the same commit.
  prerelease_suffix: "-"

  # Tags to be ignored by GoReleaser.
  # This means that GoReleaser will not pick up tags that match any of the
  # provided values as either previous or current tags.
  #
  # Templates: allowed.
  ignore_tags:
    - nightly
    - "{{.Env.IGNORE_TAG}}"

  # Tags that begin with these prefixes will be ignored.
  #
  # This feature is only available in GoReleaser Pro.
  # Templates: allowed.
  ignore_tag_prefixes:
    - foo/
    - "{{.Env.IGNORE_TAG_PREFIX}}/bar"
```

## Semver sorting

This allows you to sort tags by semver:

```yaml
# .goreleaser.yml
git:
  tag_sort: semver
```

It'll parse all tags, ignoring non-semver-compatible tags, and sort from newest
to oldest, so the latest tag is returned.

This has the effect of sorting non-pre-release tags before pre-release ones,
which is different from what other git sorting options might give you.

{% include-markdown "../includes/pro.md" comments=false %}
