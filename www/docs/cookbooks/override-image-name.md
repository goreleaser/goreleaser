# Override hardcoded registry and image name ?

If you are curious about how to avoid hard-coded registry and image names within your `.goreleaser.yml`, we have good news for you. Here is the solution. Suppose you haven't been aware of GoReleaser Action, which allows you to install GoReleaser binary in your workflow easily. In that case, this is the right time to be mindful of that because, in this section, we'll give an example through GoReleaser's GitHub Action.

> To get more detail about the GoReleaser's GitHub Action, please [see](https://github.com/goreleaser/goreleaser-action).

As you can see from the description [here](https://github.com/goreleaser/goreleaser-action#environment-variables), you can pass environment variables to the GoReleaser to use within the `.goreleaser.yml` via syntax `{{ .Env.<something> }}`. So, let' define our registry and image names as an [environment variable in the workflow](https://docs.github.com/en/actions/learn-github-actions/environment-variables), then pass those to the GoReleaser via `env` section of the GoReleaser's GitHub Action like the following:

```YAML
 jobs:
  # use goreleaser to cross-compile go binaries and add them to GitHub release
  goreleaser:
    runs-on: ubuntu-latest
    env:
      REGISTRY: "ghcr.io"
      IMAGE_NAME: "google/addlicense"
...
      - name: Run GoReleaser
        uses:  goreleaser/goreleaser-action@v2
        with:
          distribution: goreleaser
          version: latest
          args: release --rm-dist
       env:
         REGISTRY: ${{ env.REGISTRY }}
         IMAGE: ${{ env.IMAGE_NAME }}
         GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Once you pass those to the GoReleaser, you can access them within your `.goreleaser.yml` file as I mentioned above, here is the example of this:

```YAML
dockers:
    - image_templates:
        - '{{ .Env.REGISTRY }}/{{ .Env.IMAGE_NAME }}:{{ .Tag }}-amd64'
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

That's all we need to do, you even might be surprised when you notice that how easy it is to overcome this issue.
