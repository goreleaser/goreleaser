# Announce

GoReleaser can also announce new releases on social networks, chat rooms and via
email!

It runs at the very end of the pipeline and can be skipped with the
`--skip=announce` flag of the [`release`](../../cmd/goreleaser_release.md)
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

- :simple-bluesky: [Bluesky](./bluesky.md)
- :simple-discord: [Discord](./discord.md)
- :material-linkedin: [LinkedIn](./linkedin.md)
- :simple-mastodon: [Mastodon](./mastodon.md)
- :simple-mattermost: [Mattermost](./mattermost.md)
- :simple-opencollective: [OpenCollective](./opencollective.md)
- :simple-reddit: [Reddit](./reddit.md)
- :simple-slack: [Slack](./slack.md)
- :material-email: [Email/SMTP](./smtp.md)
- :material-microsoft-teams: [Teams](./teams.md)
- :simple-telegram: [Telegram](./telegram.md)
- :fontawesome-brands-x-twitter: [𝕏/Twitter](./twitter.md)
- :material-webhook: [Webhooks](./webhook.md)

</div>
