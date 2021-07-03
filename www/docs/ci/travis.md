# Travis CI

You may want to setup your project to auto-deploy your new tags on
[Travis](https://travis-ci.org), for example:

```yaml
# .travis.yml
language: go

# needed only if you use the snap pipe:
addons:
  apt:
    packages:
    - snapcraft

# needed only if you use the Docker pipe
services:
- docker

script:
  - go test ./... # replace this with your test script
  - curl -sfL https://git.io/goreleaser | sh -s -- check # check goreleaser config for deprecations

after_success:
# Docker login is required if you want to push Docker images.
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
