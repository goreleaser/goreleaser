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
    # needed only if you use the snap pipe:
    - snapcraft

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

Here is how to do it with [CircleCI](https://circleci.com):

```yml
# .circleci/config.yml
version: 2.1
workflows:
  main:
    jobs:
      - release:
          # Only run this job on git tag pushes
          filters:
            branches:
              ignore: /.*/
            tags:
              only: /v[0-9]+(\.[0-9]+)*(-.*)*/
jobs:
  release:
    docker:
      - image: circleci/golang:1.14
    steps:
      - checkout
      - run: curl -sL https://git.io/goreleaser | bash
```

## Drone

By default, drone does not fetch tags. `plugins/git` is used with default values,
in most cases we'll need overwrite the `clone` step enabling tags in order to make
`goreleaser` work correctly.

In this example we're creating a new release every time a new tag is pushed.
Note that you'll need to enable `tags` in repo settings and add `github_token`
secret.

#### 1.x
```yml
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

CloudBuild works off a different clone than your GitHub repo: it seems that
your changes are pulled to a repo like
`source.developers.google.com/p/YourProjectId/r/github-YourGithubUser-YourGithubRepo`,
and that's what you're building off.

This repo has the wrong name, so to prevent GoReleaser from publishing to
the wrong GitHub repo, add to your `.goreleaser.yml` file's release section:

```yml
release:
  github:
    owner: YourGithubUser
    name: YourGithubRepo
```

Create two build triggers:

- a "push to any branch" trigger for your regular CI (doesn't invoke GoReleaser)
- a "push to tag" trigger which invokes GoReleaser

The push to any branch trigger could use a `Dockerfile` or a `cloudbuild.yaml`,
whichever you prefer.

You should have a dedicated `cloudbuild.release.yaml` that is only used by the
"push to tag" trigger.

In this example we're creating a new release every time a new tag is pushed.
See [Using Encrypted Resources](https://cloud.google.com/cloud-build/docs/securing-builds/use-encrypted-secrets-credentials)
for how to encrypt and base64-encode your github token.

The clone that the build uses
[has no tags](https://issuetracker.google.com/u/1/issues/113668706),
which is why we must explicitly run `git tag $TAG_NAME` (note that `$TAG_NAME`
is only set when your build is triggered by a "push to tag".)
This will allow GoReleaser to create a release with that version,
but it won't be able to build a proper
changelog containing just the messages from the commits since the prior tag.
Note that the build performs a shallow clone of git repositories and will
only contain tags that reference the latest commit.

```yml
steps:
# Setup the workspace so we have a viable place to point GOPATH at.
- name: gcr.io/cloud-builders/go
  env: ['PROJECT_ROOT=github.com/YourGithubUser/YourGithubRepo']
  args: ['env']

# Create github release.
- name: goreleaser/goreleaser
  entrypoint: /bin/sh
  dir: gopath/src/github.com
  env: ['GOPATH=/workspace/gopath']
  args: ['-c', 'cd YourGithubUser/YourGithubRepo && git tag $TAG_NAME && /goreleaser' ]
  secretEnv: ['GITHUB_TOKEN']

  secrets:
  - kmsKeyName: projects/YourProjectId/locations/global/keyRings/YourKeyRing/cryptoKeys/YourKey
    secretEnv:
      GITHUB_TOKEN: |
        ICAgICAgICBDaVFBZUhVdUVoRUtBdmZJSGxVWnJDZ0hOU2NtMG1ES0k4WjF3L04zT3pEazhRbDZr
        QVVTVVFEM3dVYXU3cVJjK0g3T25UVW82YjJaCiAgICAgICAgREtBMWVNS0hOZzcyOUtmSGoyWk1x
        ICAgICAgIEgwYndIaGUxR1E9PQo=

```

## Semaphore

In [Sempahore 2.0](https://semaphoreci.com) each project starts with the
default pipeline specified in `.semaphore/semaphore.yml`.

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

Pipeline file in `.semaphore/goreleaser.yml`:

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

The following YAML file, `createSecret.yml` creates a new secret item that is
called GoReleaser with one environment variable, named `GITHUB_TOKEN`:

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

Check [Managing Secrets](https://docs.semaphoreci.com/article/51-secrets-yaml-reference)
for more detailed documentation.

## GitLab CI

To create GitLab releases and push images to a Docker registry, add a file
`.gitlab-ci.yml` to the root of the project:

```yaml
stages:
  - release

release:
  stage: release
  image: docker:stable
  services:
    - docker:dind

  variables:
    GORELEASER_IMAGE: goreleaser/goreleaser:latest

    # Optionally use GitLab's built-in image registry.
    # DOCKER_REGISTRY: $CI_REGISTRY
    # DOCKER_USERNAME: $CI_REGISTRY_USER
    # DOCKER_PASSWORD: $CI_REGISTRY_PASSWORD

    # Or, use any registry, including the official one.
    DOCKER_REGISTRY: https://index.docker.io/v1/

    # Disable shallow cloning so that goreleaser can diff between tags to
    # generate a changelog.
    GIT_DEPTH: 0

  # Only run this release job for tags, not every commit (for example).
  only:
    refs:
      - tags

  script: |
    docker pull $GORELEASER_IMAGE

    # GITLAB_TOKEN is needed to create GitLab releases.
    # DOCKER_* are needed to push Docker images.
    docker run --pull --rm --privileged \
      -v $PWD:/go/src/gitlab.com/YourGitLabUser/YourGitLabRepo \
      -w /go/src/gitlab.com/YourGitLabUser/YourGitLabRepo \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e DOCKER_USERNAME -e DOCKER_PASSWORD -e DOCKER_REGISTRY  \
      -e GITLAB_TOKEN \
      $GORELEASER_IMAGE release --rm-dist
```

In GitLab CI settings, add variables for `DOCKER_REGISTRY`, `DOCKER_USERNAME`,
and `DOCKER_PASSWORD` if you aren't using the GitLab image registry. If you are
using the GitLab image registry, you don't need to set these.

Add a variable `GITLAB_TOKEN` if you are using [GitLab
releases](https://docs.gitlab.com/ce/user/project/releases/). The value should
be an API token with `api` scope for a user that has access to the project.

The secret variables, `DOCKER_PASSWORD` and `GITLAB_TOKEN`, should be masked.
Optionally, you might want to protect them if the job that uses them will only
be run on protected branches or tags.

Make sure the `image_templates` in the file `.goreleaser.yml` reflect that
custom registry!

Example:

```yaml
dockers:
-
  goos: linux
  goarch: amd64
  binaries:
  - program
  image_templates:
  - 'registry.gitlab.com/Group/Project:{{ .Tag }}'
  - 'registry.gitlab.com/Group/Project:latest'
```

## Codefresh

Codefresh uses Docker based pipelines where all steps must be Docker containers.
Using GoReleaser is very easy via the
[existing Docker image](https://hub.docker.com/r/goreleaser/goreleaser/).

Here is an example pipeline that builds a Go application and then uses
GoReleaser.

```yaml
version: '1.0'
stages:
  - prepare
  - build
  - release
steps:
  main_clone:
    title: 'Cloning main repository...'
    type: git-clone
    repo: '${{CF_REPO_OWNER}}/${{CF_REPO_NAME}}'
    revision: '${{CF_REVISION}}'
    stage: prepare
  BuildMyApp:
    title: Compiling go code
    stage: build
    image: 'golang:1.14'
    commands:
      - go build
  ReleaseMyApp:
    title: Creating packages
    stage: release
    image: 'goreleaser/goreleaser'
    commands:
      - goreleaser --rm-dist
```

You need to pass the variable `GITHUB_TOKEN` in the Codefresh UI that
contains credentials to your Github account or load it from
[shared configuration](https://codefresh.io/docs/docs/configure-ci-cd-pipeline/shared-configuration/).
You should also restrict this pipeline to run only on tags when you add
[git triggers](https://codefresh.io/docs/docs/configure-ci-cd-pipeline/triggers/git-triggers/)
on it.

More details can be found in the
[GoReleaser example page](https://codefresh.io/docs/docs/learn-by-example/golang/goreleaser/).
