---
title: "Retries"
weight: 115
---

{{< version "v2.15.3-unreleased" >}}

Everything that does network calls go through a retry mechanism, so it will do a
back-off retry on failures deemed _retriable_.

The configuration is as follows:

```yaml {filename=".goreleaser.yml"}
retry:
  # Set max retry count.
  # Setting to 0 will retry until the retried function succeeds.
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
