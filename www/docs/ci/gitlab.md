# GitLab CI

Below are some example GitLab CI jobs that use GoReleaser to release a project.

> If you are using private hosted or Enterprise version of Gitlab, please follow this [guide](/scm/gitlab/) before diving into the details.

## Basic Releasing

You can easily run GoReleaser in GitLab CI using its Docker container.

In the repository's GitLab CI settings, add a `GITLAB_TOKEN` variable. The value should
be an API token with `api` scope for a user that has access to the project. This
variable should be masked and optionally protected if the job will only run on
protected branches and tags.
See [Quick Start](https://goreleaser.com/quick-start/) for more information on
GoReleaser's environment variables.

Add a `.gitlab-ci.yml` file to the root of the project:

```yaml
stages:
  - release

release:
  stage: release
  image:
    name: goreleaser/goreleaser
    entrypoint: ['']
  only:
    - tags
  variables:
    # Disable shallow cloning so that goreleaser can diff between tags to
    # generate a changelog.
    GIT_DEPTH: 0
  script:
    - goreleaser release --rm-dist
```

Notice that `entrypoint` is intentionally blank. See the
[GitLab documentation on entrypoints](https://docs.gitlab.com/ee/ci/docker/using_docker_images.html#overriding-the-entrypoint-of-an-image)
for more information.

When tags are pushed to the repository,
an available GitLab Runner with the Docker executor will pick up the release job.
`goreleaser/goreleaser` will start in a container and the repository will be mounted inside.
Finally, the `script` section will run within the container starting in your project's directory.

## Releasing Archives and Pushing Images

Pushing images to a registry requires using Docker-in-Docker. To create GitLab releases and push
images to a Docker registry, add a file `.gitlab-ci.yml` to the root of the project:

```yaml
stages:
  - release

release:
  stage: release
  image: docker:stable
  services:
    - docker:dind

  variables:
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
    # GITLAB_TOKEN is needed to create GitLab releases.
    # DOCKER_* are needed to push Docker images.
    docker run --rm --privileged \
      -v $PWD:/go/src/gitlab.com/YourGitLabUser/YourGitLabRepo \
      -w /go/src/gitlab.com/YourGitLabUser/YourGitLabRepo \
      -v /var/run/docker.sock:/var/run/docker.sock \
      -e DOCKER_USERNAME -e DOCKER_PASSWORD -e DOCKER_REGISTRY  \
      -e GITLAB_TOKEN \
      goreleaser/goreleaser release --rm-dist
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

Make sure the `image_templates` in the file `.goreleaser.yaml` reflect that
custom registry!

Example:

```yaml
dockers:
-
  goos: linux
  goarch: amd64
  image_templates:
  - 'registry.gitlab.com/Group/Project:{{ .Tag }}'
  - 'registry.gitlab.com/Group/Project:latest'
```

## Example Repository

You can check [this example repository](https://gitlab.com/goreleaser/example) for a real world example.

<a href="https://gitlab.com/goreleaser/example/-/releases">
  <figure>
    <img src="https://img.carlosbecker.dev/goreleaser-gitlab.png"/>
    <figcaption>Example release on GitLab.</figcaption>
  </figure>
</a>
