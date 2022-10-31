# Woodpecker

By default, woodpecker only fetches tags on `tag` events. If you are not using a tag event, you will need to override the git plugin like so:

```yaml
clone:
  git:
    image: woodpeckerci/plugin-git
    settings:
      tags: true
```

Here is how to set up a basic release pipeline with [Woodpecker](https://woodpecker-ci.org) and [Gitea](https://gitea.io).

```yaml
pipeline:
  release:
    image: goreleaser/goreleaser
    commands:
      - goreleaser release
    secrets: [ gitea_token ]
    when:
      event: tag
```
