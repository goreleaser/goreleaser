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
    volumes:
      - name: deps
        path: /go
    commands:
      - go test -race -v ./... -cover

  - name: release
    image: golang
    environment:
      GITHUB_TOKEN:
        from_secret: github_token
    volumes:
      - name: deps
        path: /go
    commands:
      - curl -sL https://git.io/goreleaser | bash
    when:
      event: tag

volumes:
  - name: deps
    temp: {}
```

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

