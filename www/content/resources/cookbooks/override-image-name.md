---
title: "Override hardcoded registry and image name?"
weight: 140
---

To avoid hard-coding the registry and image name in your `.goreleaser.yaml`,
pass them as environment variables. This recipe shows how to do that with the
[GoReleaser GitHub Action](https://github.com/goreleaser/goreleaser-action).

> For more details, see the
> [GoReleaser GitHub Action documentation](https://github.com/goreleaser/goreleaser-action).

The [GoReleaser GitHub Action](https://github.com/goreleaser/goreleaser-action#environment-variables)
lets you pass environment variables that GoReleaser can read in
`.goreleaser.yaml` with the `{{ .Env.<name> }}` syntax. Define the registry and
image name as [workflow environment variables](https://docs.github.com/en/actions/learn-github-actions/environment-variables),
then pass them to GoReleaser through the action's `env` section:

```yaml {filename=".github/workflows/release.yml"}
jobs:
  # use goreleaser to cross-compile go binaries and add them to GitHub release
  goreleaser:
    runs-on: ubuntu-latest
    env:
      REGISTRY: "ghcr.io"
      IMAGE_NAME: "google/addlicense"
    steps:
      # ...
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v7
        with:
          distribution: goreleaser
          version: latest
          args: release --clean
        env:
          REGISTRY: ${{ env.REGISTRY }}
          IMAGE_NAME: ${{ env.IMAGE_NAME }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Now you can reference them in your `.goreleaser.yaml`:

```yaml {filename=".goreleaser.yaml"}
dockers:
  - image_templates:
      - "{{ .Env.REGISTRY }}/{{ .Env.IMAGE_NAME }}:{{ .Tag }}-amd64"
    dockerfile: Dockerfile.goreleaser
    use: buildx
    build_flag_templates:
      - "--pull"
      - "--label=org.opencontainers.image.created={{.Date}}"
      - "--label=org.opencontainers.image.name={{.ProjectName}}"
      - "--label=org.opencontainers.image.revision={{.FullCommit}}"
      - "--label=org.opencontainers.image.version={{.Version}}"
      - "--label=org.opencontainers.image.source={{.GitURL}}"
      - "--platform=linux/amd64"
```
