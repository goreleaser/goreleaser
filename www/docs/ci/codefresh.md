# Codefresh

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
    image: 'golang:1.16'
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
