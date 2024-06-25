# Closing Milestones

GoReleaser can close repository milestones after successfully publishing all
artifacts.

Let's see what can be customized in the `milestones` section:

```yaml
# .goreleaser.yaml
milestones:
  # You can have multiple milestone configs
  - # Repository for the milestone
    #
    # Default: extracted from the origin remote URL.
    repo:
      owner: user
      name: repo

    # Whether to close the milestone
    close: true

    # Fail release on errors, such as missing milestone.
    fail_on_error: true

    # Name of the milestone
    #
    # Default: '{{ .Tag }}'.
    name_template: "Current Release"
```

{% include-markdown "../includes/templates.md" comments=false %}
