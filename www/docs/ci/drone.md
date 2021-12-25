# Drone

By default, drone does not fetch tags. `plugins/git` is used with default values,
in most cases we'll need overwrite the `clone` step enabling tags in order to make
`goreleaser` work correctly.

In this example we're creating a new release every time a new tag is pushed.
Note that you'll need to enable `tags` in repo settings and add `github_token`
secret.

#### 1.x
```yaml
# .drone.yml

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

In case you need to build docker image, use [Docker-In-Docker](https://docs.drone.io/pipeline/docker/examples/services/docker_dind/) (DIND)

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

Note: to use DIND you have to set repo as 'trusted'. To mark repository as trusted:

1. contact your Drone's admin
2. or set your [user as administrator](https://docs.drone.io/server/user/admin/) and then enable 'trusted' switch in repository settings UI

#### 0.8
```yaml
pipeline:
  clone:
    image: plugins/git
    tags: true

  test:
    image: golang:1.10
    commands:
      - go test ./... -race

  release:
    image: golang:1.10
    secrets: [github_token]
    commands:
      curl -sL https://git.io/goreleaser | bash
    when:
      event: tag
```

