---
title: Continuous Integration
menu: true
weight: 140
---

GoReleaser was built from the very first commit with the idea of
running it as part of the CI pipeline in mind.

Let's see how we can get it working on popular CI softwares.

## Travis

You may want to setup your project to auto-deploy your new tags on
[Travis](https://travis-ci.org), for example:

```yaml
# .travis.yml
language: go

addons:
  apt:
    packages:
    # needed for the nfpm pipe:
    - rpm
    # needed for the snap pipe:
    - snapcraft

env:
# needed for the snap pipe:
- PATH=/snap/bin:$PATH

install:
# needed for the snap pipe:
- sudo snap install snapcraft --classic

# needed for the docker pipe
services:
- docker

after_success:
# docker login is required if you want to push docker images.
# DOCKER_PASSWORD should be a secret in your .travis.yml configuration.
- test -n "$TRAVIS_TAG" && docker login -u=myuser -p="$DOCKER_PASSWORD"

# calls goreleaser
deploy:
- provider: script
  skip_cleanup: true
  script: curl -sL https://git.io/goreleaser | bash
  on:
    tags: true
    condition: $TRAVIS_OS_NAME = linux
```

Note the last line (`condition: $TRAVIS_OS_NAME = linux`): it is important
if you run a build matrix with multiple Go versions and/or multiple OSes. If
that's the case you will want to make sure GoReleaser is run just once.

# Circle

Here is how to do it with [CircleCI](https://circleci.com):

```yml
# circle.yml
deployment:
  tag:
    tag: /v[0-9]+(\.[0-9]+)*(-.*)*/
    owner: user
    commands:
      - curl -sL https://git.io/goreleaser | bash
```

## Drone

By default, drone does not fetch tags. `plugins/git` is used with default values,
in most cases we'll need overwrite the `clone` step enabling tags in order to make
`goreleaser` work correctly.

In this example we're creating a new release every time a new tag is pushed.
Note that you'll need to enable `tags` in repo settings and add `github_token`
secret.

```yml
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
