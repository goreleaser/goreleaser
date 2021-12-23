# Announce

GoReleaser can also announce new releases on social networks, chat rooms and via email!

It runs at the very end of the pipeline and can be skipped with the `--skip-announce` flag of the [`release`](/cmd/goreleaser_release/) command, or via the skip property:

```yaml
# .goreleaser.yaml
announce:
  # Skip the announcing feature in some conditions, for instance, when publishing patch releases.
  # Valid options are `true`, `false`, empty, or a template that evaluates to a boolean (`true` or `false`).
  # Defaults to empty (which means false).
  skip: "{{gt .Patch 0}}"
```
