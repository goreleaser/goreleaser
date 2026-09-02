---
title: "Drone"
weight: 70
---

By default, Drone does not fetch tags. `plugins/git` runs with default
values, so in most cases you need to override the `clone` step to enable tags
for `goreleaser` to work correctly.

In this example we're creating a new release every time a new tag is pushed.
Note that you'll need to enable `tags` in repo settings and add `github_token`
secret.

#### 1.x
```yaml {filename=".drone.yml"}
kind: pipeline
name: default

steps:
  - name: fetch
    image: docker:git
    commands:
      - git fetch --tags

  - name: test
    image: golang
    commands:
      - go test -race -v ./... -cover

  - name: release
    image: goreleaser/goreleaser
    environment:
      GITHUB_TOKEN:
        from_secret: github_token
    commands:
      - goreleaser release
    when:
      event: tag
```

In case you need to build a Docker image, use [Docker-in-Docker](https://docs.drone.io/pipeline/docker/examples/services/docker_dind/) (DinD)

```yaml
---
kind: pipeline
name: default
trigger:
  ref:
    - refs/tags/*

services:
  - name: docker
    image: docker:dind
    privileged: true
    volumes:
      - name: dockersock
        path: /var/run

steps:
  - name: fetch
    image: docker:git
    commands:
      - git fetch --tags

  - name: test
    image: golang
    commands:
      - go test -race -v ./... -cover

  - name: release
    image: goreleaser/goreleaser
    environment:
      GITHUB_TOKEN:
        from_secret: github_token
    volumes:
      - name: dockersock
        path: /var/run
    commands:
      - goreleaser release
    when:
      event: tag

volumes:
  - name: dockersock
    temp: {}
```

Note: to use DinD you have to set the repository as 'trusted'. To mark a
repository as trusted:

1. contact your Drone administrator
2. or set your [user as administrator](https://docs.drone.io/server/user/admin/) and then enable 'trusted' switch in repository settings UI

#### 0.8
```yaml
pipeline:
  clone:
    image: plugins/git
    tags: true

  test:
    image: golang:1.27
    commands:
      - go test ./... -race

  release:
    image: golang:1.27
    secrets: [github_token]
    commands:
      curl -sfL https://goreleaser.com/static/run | bash
    when:
      event: tag
```

