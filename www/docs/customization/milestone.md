---
title: Milestone
---

GoReleaser can close repository milestones after successfully
publishing all artifacts.

Let's see what can be customized in the `milestones` section:

```yaml
# .goreleaser.yml
milestones:
  # You can have multiple milestone configs
  -
    # Repository for the milestone
    # Default is extracted from the origin remote URL
    repo:
      owner: user
      name: repo

    # Whether to close the milestone
    # Default is false
    close: true

    # Name of the milestone
    # Default is `{{ .Tag }}`
    name_template: "Current Release"
```

!!! tip
    Learn more about the [name template engine](/customization/templates).
