# Cirrus CI

Here is how to do it with [Cirrus CI](https://cirrus-ci.org):

```yaml
# .cirrus.yml
task:
  name: Release
  only_if: $CIRRUS_TAG != '' # run only on tags
  depends_on:
    - Test
    - Lint
    # any other sanity tasks
  env:
    GITHUB_TOKEN: ENCRYPTED[ABC]
  container:
    image: goreleaser/goreleaser:latest
  release_script: goreleaser
```

**Note:** you'll need to create an [encrypted variable](https://cirrus-ci.org/guide/writing-tasks/#encrypted-variables)
to store `GITHUB_TOKEN` for GoReleaser to access GitHub API.
