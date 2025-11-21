# Git

This allows you to change the behavior of some Git commands.

```yaml title=".goreleaser.yaml"
git:
  # What should be used to sort tags when gathering the current and previous
  # tags if there are more than one tag in the same commit.
  #
  # See: https://git-scm.com/docs/git-tag#Documentation/git-tag.txt---sortltkeygt
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

<!-- md:featpro -->

This allows you to sort tags by semver:

```yaml title=".goreleaser.yaml"
git:
  tag_sort: semver
```

It'll parse all tags, ignoring non-semver-compatible tags, and sort from newest
to oldest, so the latest tag is returned.

This has the effect of sorting non-pre-release tags before pre-release ones,
which is different from what other git sorting options might give you.

## Smart semver sorting

<!-- md:version v2.12 -->

<!-- md:experimental -->

<!-- md:featpro -->

Like semver sorting, but smarter: if the current version is not a pre-release,
it'll search for previous tags that are not pre-releases.

Imagine you have a history like this:

```
v0.1.0
v0.2.0-beta.1
v0.2.0-beta.2
v0.2.0-beta.3
v0.2.0
```

And you want to release `v0.2.0`.
Usually, GoReleaser would get `v0.2.0-beta.3` as previous version, but that's
likely not what most people would expect (`v0.1.0`).
Smart semver will ignore pre-release versions in these cases, making the release
notes more complete.

If you were to release `v0.2.0-beta.3`, though, it would still get
`v0.2.0-beta.2` as previous version, which I think makes sense.

To use it, add this to your configuration:

```yaml title=".goreleaser.yaml"
git:
  tag_sort: smartsemver
```
