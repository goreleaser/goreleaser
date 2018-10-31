---
title: Continuous Integration
menu: true
weight: 140
---

GoReleaser was built from the very first commit with the idea of
running it as part of the CI pipeline in mind.

Let's see how we can get it working on popular CI software.

## Travis CI

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
# snapcraft login is required if you want to push snapcraft packages to the
# store.
# You'll need to run `snapcraft export-login snap.login` and
# `travis encrypt-file snap.login --add` to add the key to the travis
# environment.
- test -n "$TRAVIS_TAG" && snapcraft login --with snap.login

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

## CircleCI

Here is how to do it with [CircleCI 2.0](https://circleci.com):

```yml
# .circleci/config.yml
version: 2
jobs:
  release:
    docker:
      - image: circleci/golang:1.10
    steps:
      - checkout
      - run: curl -sL https://git.io/goreleaser | bash
workflows:
  version: 2
  release:
    jobs:
      - release:
          filters:
            branches:
              ignore: /.*/
            tags:
              only: /v[0-9]+(\.[0-9]+)*(-.*)*/
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

## Google CloudBuild

CloudBuild works off a different clone than your github repo: it seems that
your changes are pulled to a repo like
source.developers.google.com/p/YourProjectId/r/github-YourGithubUser-YourGithubRepo, and that's what
you're building off.

This repo has the wrong name, so to prevent Goreleaser from publishing to
the wrong github repo, put in the your .goreleaser.yml file's release section:

```yml
release:
  github:
    owner: YourGithubUser
    name: YourGithubRepo
```

Create two build triggers:
- a "push to any branch" trigger for your regular CI (doesn't invoke goreleaser)
- a "push to tag" trigger which invokes goreleaser

The push to any branch trigger could use a Dockerfile or a cloudbuild.yaml,
whichever you prefer.

You should have a dedicated cloudbuild.release.yaml that is only used by the "push to
tag" trigger.

In this example we're creating a new release every time a new tag is pushed.
See [Using Encrypted Resources](https://cloud.google.com/cloud-build/docs/securing-builds/use-encrypted-secrets-credentials) for how to encrypt and base64-encode your github token.

The clone that the build uses [has no
tags](https://issuetracker.google.com/u/1/issues/113668706), which is why we
must explicitly run git tag $TAG_NAME (note that $TAG_NAME is only set when
your build is triggered by a "push to tag".) This will allow goreleaser to
create a release with that version, but it won't be able to build a proper
changelog containing just the messages from the commits since the prior tag.

```yml
steps:
~ # Setup the workspace so we have a viable place to point GOPATH at.
~ - name: gcr.io/cloud-builders/go
~   env: ['PROJECT_ROOT=github.com/YourGithubUser/YourGithubRepo']
~_  args: ['env']

~ # Create github release.
~ - name: goreleaser/goreleaser
~   entrypoint: /bin/sh
~   dir: gopath/src/github.com
~   env: ['GOPATH=/workspace/gopath']
~   args: ['-c', 'cd YourGithubUser/YourGithubRepo && git tag $TAG_NAME && /goreleaser' ]
~_  secretEnv: ['GITHUB_TOKEN']

  secrets:
~ - kmsKeyName: projects/YourProjectId/locations/global/keyRings/YourKeyRing/cryptoKeys/YourKey
~   secretEnv:
~     GITHUB_TOKEN: |
~       ICAgICAgICBDaVFBZUhVdUVoRUtBdmZJSGxVWnJDZ0hOU2NtMG1ES0k4WjF3L04zT3pEazhRbDZr
~       QVVTVVFEM3dVYXU3cVJjK0g3T25UVW82YjJaCiAgICAgICAgREtBMWVNS0hOZzcyOUtmSGoyWk1x
~_      ICAgICAgIEgwYndIaGUxR1E9PQo=

```

## Semaphore

In [Sempahore 2.0](https://semaphoreci.com) each project starts with the
default pipeline specified in .semaphore/semaphore.yml.

```yml
# .semaphore/semaphore.yml.
version: v1.0
name: Build
agent:
  machine:
    type: e1-standard-2
    os_image: ubuntu1804

blocks:
  - name: "Test"
    task:
      prologue:
        commands:
          # set go version
          - sem-version go 1.11
          - "export GOPATH=~/go"
          - "export PATH=/home/semaphore/go/bin:$PATH"
          - checkout

      jobs:
        - name: "Lint"
          commands:
            - go get ./...
            - go test ./...

# On Semaphore 2.0 deployment and delivery is managed with promotions,
# which may be automatic or manual and optionally depend on conditions.
promotions:
    - name: Release
       pipeline_file: goreleaser.yml
       auto_promote_on:
         - result: passed
           branch:
             - "^refs/tags/v*"
```

Pipeline file in .semaphore/goreleaser.yml:

```yml
version: "v1.0"
name: GoReleaser
agent:
  machine:
    type: e1-standard-2
    os_image: ubuntu1804
blocks:
  - name: "Release"
    task:
      secrets:
        - name: goreleaser
      prologue:
        commands:
          - sem-version go 1.11
          - "export GOPATH=~/go"
          - "export PATH=/home/semaphore/go/bin:$PATH"
          - checkout
      jobs:
      - name: goreleaser
        commands:
          - curl -sL https://git.io/goreleaser | bash
```

The following YAML file, `createSecret.yml` creates a new secret item that is called goreleaser
with one environment variable, named GITHUB_TOKEN:

```yml
apiVersion: v1alpha
kind: Secret
metadata:
  name: goreleaser
data:
  env_vars:
    - name: GITHUB_TOKEN
      value: "4afk4388304hfhei34950dg43245"
```

Check [Managing Secrets](https://docs.semaphoreci.com/article/15-secrets) for
more detailed documentation.
