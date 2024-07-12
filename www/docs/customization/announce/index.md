# Announce

GoReleaser can also announce new releases on social networks, chat rooms and via
email!

It runs at the very end of the pipeline and can be skipped with the
`--skip=announce` flag of the [`release`](../../cmd/goreleaser_release.md)
command, or via the skip property:

```yaml
# .goreleaser.yaml
announce:
  # Skip the announcing feature in some conditions, for instance, when
  # publishing patch releases.
  #
  # Any value different from 'true' is evaluated to false.
  #
  # Templates: allowed.
  skip: "{{gt .Patch 0}}"
```
