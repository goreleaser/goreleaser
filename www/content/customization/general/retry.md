---
title: "Retries"
weight: 115
---

{{< g_version "v2.15.3" >}}

Most external service calls go through a retry mechanism with exponential
back-off on failures deemed _retriable_.

This includes:

- **Git providers** — GitHub, GitLab, and Gitea API calls (releases, uploads,
  milestones, pull requests, etc.)
- **Announcement pipes** — Discord, Telegram, Slack, Mastodon, Teams, Reddit,
  Twitter, Bluesky, LinkedIn, Discourse, Mattermost, Webhook, OpenCollective,
  and MCP
- **HTTP uploads** — Artifactory, custom HTTP uploads, and similar

Transient failures (network errors, HTTP 5xx, and 429 Too Many Requests) are
automatically retried. Permanent failures (4xx, file-not-found, etc.) are not.

The configuration is as follows:

```yaml {filename=".goreleaser.yml"}
retry:
  # Set max retry count.
  # Setting to 1 disables retries (single attempt).
  #
  # Default: 10
  attempts: 15

  # Set delay between retry
  #
  # Default: 10s
  delay: 10s

  # Default: 5m
  max_delay: 3m
```
