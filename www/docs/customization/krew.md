# Krew Plugin Manifests

After releasing to GitHub or GitLab, GoReleaser can generate and publish a _Krew
Plugin Manifest_ into a repository that you have access to.

Check their [website](https://krew.sigs.k8s.io) for more information.

The `krews` section specifies how the plugins should be created:

```yaml
# .goreleaser.yaml
krews:
  -
    # Name of the recipe
    #
    # Default: ProjectName
    name: myproject

    # IDs of the archives to use.
    ids:
    - foo
    - bar

    # GOARM to specify which 32-bit arm version to use if there are multiple
    # versions from the build section. Krew plugin supports at this moment only
    # one 32-bit version.
    #
    # Default: 6
    goarm: 6

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: 'v1'
    goamd64: v3

    # NOTE: make sure the url_template, the token and given repo (github or
    # gitlab) owner and name are from the same kind. We will probably unify this
    # in the next major version like it is done with scoop.

    # GitHub/GitLab repository to push the Krew plugin to
    # Gitea is not supported yet, but the support coming
    index:
      # Repository owner.
      #
      # Templates: allowed
      owner: user

      # Repository name.
      #
      # Templates: allowed
      name: krew-plugins

      # Optionally a branch can be provided.
      #
      # Default: default repository branch
      # Templates: allowed
      branch: main

      # Optionally a token can be provided, if it differs from the token
      # provided to GoReleaser
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"

      # Sets up pull request creation instead of just pushing to the given branch.
      # Make sure the 'branch' property is different from base before enabling
      # it.
      #
      # Since: v1.17
      pull_request:
        # Whether to enable it or not.
        enabled: true

        # Base branch of the PR.
        #
        # Default: default repository branch.
        base: main

      # Clone, create the file, commit and push, to a regular Git repository.
      #
      # Notice that this will only have any effect if the given URL is not
      # empty.
      #
      # Since: v1.18
      git:
        # The Git URL to push.
        url: 'ssh://git@myserver.com:repo.git'

        # The SSH private key that should be used to commit to the Git
        # repository.
        # This can either be a path or the key contents.
        #
        # IMPORTANT: the key must not be password-protected.
        #
        # WARNING: do not expose your private key in the configuration file!
        private_key: '{{ .Env.PRIVATE_KEY_PATH }}'

        # The value to be passed to `GIT_SSH_COMMAND`.
        # This is mainly used to specify the SSH private key used to pull/push
        # to the Git URL.
        #
        # Default: 'ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null'
        ssh_command: 'ssh -i {{ .Env.KEY }} -o SomeOption=yes'

    # URL which is determined by the given Token (github or
    # gitlab)
    # Default:
    #   GitHub: 'https://github.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}'
    #   GitLab: 'https://gitlab.com/<repo_owner>/<repo_name>/-/releases/{{ .Tag }}/downloads/{{ .ArtifactName }}'
    #   Gitea: 'https://gitea.com/<repo_owner>/<repo_name>/releases/download/{{ .Tag }}/{{ .ArtifactName }}'
    url_template: "http://github.mycompany.com/foo/bar/releases/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    commit_msg_template: "Krew plugin update for {{ .ProjectName }} version {{ .Tag }}"

    # Your app's homepage.
    homepage: "https://example.com/"

    # Your app's description.
    # The usual guideline for this is to wrap the line at 80 chars.
    #
    # Templates: allowed
    description: "Software to create fast and easy drum rolls."

    # Your app's short description.
    # The usual guideline for this is to be at most 50 chars long.
    #
    # Templates: allowed
    short_description: "Software to create fast and easy drum rolls."

    # Caveats for the user of your binary.
    # The usual guideline for this is to wrap the line at 80 chars.
    caveats: "How to use this binary"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # krew plugin - instead, the plugin file will be stored on the dist folder
    # only, leaving the responsibility of publishing it to the user.
    # If set to auto, the release will not be uploaded to the Krew plugin
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    skip_upload: true
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## Limitations

- Only one binary per archive is allowed;
- Binary releases (when `archives.format` is set to `binary`) are not allowed;
- Only one `GOARM` build is allowed;
