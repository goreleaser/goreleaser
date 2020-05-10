# Google CloudBuild

CloudBuild works off a different clone than your GitHub repo: it seems that
your changes are pulled to a repo like
`source.developers.google.com/p/YourProjectId/r/github-YourGithubUser-YourGithubRepo`,
and that's what you're building off.

This repo has the wrong name, so to prevent GoReleaser from publishing to
the wrong GitHub repo, add to your `.goreleaser.yml` file's release section:

```yaml
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

```yaml
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

