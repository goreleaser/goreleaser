---
weight: 1
---# Announce

GoReleaser can also announce new releases on social networks, chat rooms and via
email!

It runs at the very end of the pipeline and can be skipped with the
`--skip=announce` flag of the [`release`](../../cmd/goreleaser_release/)
command, or via the skip property:

```yaml title=".goreleaser.yaml"
announce:
  # Skip the announcing feature in some conditions, for instance, when
  # publishing patch releases.
  #
  # Any value different from 'true' is evaluated to false.
  #
  # Templates: allowed.
  skip: "{{gt .Patch 0}}"
```

## Supported announcers:

<div class="grid cards" markdown>

- :simple-bluesky: [Bluesky](./bluesky/)
- :simple-discord: [Discord](./discord/)
- :simple-discourse: [Discourse](./discourse/)
- :material-linkedin: [LinkedIn](./linkedin/)
- :simple-mastodon: [Mastodon](./mastodon/)
- :simple-mattermost: [Mattermost](./mattermost/)
- :simple-opencollective: [OpenCollective](./opencollective/)
- :simple-reddit: [Reddit](./reddit/)
- :simple-slack: [Slack](./slack/)
- :material-email: [Email/SMTP](./smtp/)
- :material-microsoft-teams: [Teams](./teams/)
- :simple-telegram: [Telegram](./telegram/)
- :fontawesome-brands-x-twitter: [𝕏/Twitter](./twitter/)
- :material-webhook: [Webhooks](./webhook/)

</div>
