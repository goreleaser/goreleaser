# Nixpkgs

> Since: v1.19

After releasing to GitHub, GitLab, or Gitea, GoReleaser can generate and publish
a _nixpkg_ to a [Nix User Repository][nur].

[nur]: https://github.com/nix-community/NUR

The `nix` section specifies how the pkgs should be created:

```yaml
# .goreleaser.yaml
nix:
  -
    # Name of the recipe
    #
    # Default: ProjectName
    # Templates: allowed
    name: myproject

    # IDs of the archives to use.
    # Empty means all IDs.
    ids:
    - foo
    - bar

    # GOAMD64 to specify which amd64 version to use if there are multiple
    # versions from the build section.
    #
    # Default: v1
    goamd64: v1

    # GitHub/GitLab repository to push the pkg to.
    repository:
      # Repository owner.
      #
      # Templates: allowed
      owner: user

      # Repository name.
      #
      # Templates: allowed
      name: nur

      # Optionally a branch can be provided.
      #
      # Default: default repository branch.
      # Templates: allowed
      branch: foo

      # Optionally a token can be provided, if it differs from the token
      # provided to GoReleaser
      #
      # Templates: allowed
      token: "{{ .Env.NUR_GITHUB_TOKEN }}"

      # Sets up pull request creation instead of just pushing to the given branch.
      # Make sure the 'branch' property is different from base before enabling
      # it.
      pull_request:
        # Whether to enable it or not.
        enabled: true

        # Base branch of the PR.
        # If base is a string, the PR will be opened into the same repository.
        #
        # Default: default repository branch.
        base: main

        # Base can also be another repository, in which case the owner and name
        # above will be used as HEAD, allowing cross-repository pull requests.
        #
        # Since: v1.19
        base:
          owner: org
          name: nur
          branch: main

      # Clone, create the file, commit and push, to a regular Git repository.
      #
      # Notice that this will only have any effect if the given URL is not
      # empty.
      git:
        # The Git URL to push.
        #
        # Templates: allowed
        url: 'ssh://git@myserver.com:repo.git'

        # The SSH private key that should be used to commit to the Git
        # repository.
        # This can either be a path or the key contents.
        #
        # IMPORTANT: the key must not be password-protected.
        #
        # WARNING: do not expose your private key in the configuration file!
        #
        # Templates: allowed
        private_key: '{{ .Env.PRIVATE_KEY_PATH }}'

        # The value to be passed to `GIT_SSH_COMMAND`.
        # This is mainly used to specify the SSH private key used to pull/push
        # to the Git URL.
        #
        # Default: 'ssh -i {{ .KeyPath }} -o StrictHostKeyChecking=accept-new -F /dev/null'
        # Templates: allowed
        ssh_command: 'ssh -i {{ .Env.KEY }} -o SomeOption=yes'

    # URL which is determined by the given Token (github, gitlab or gitea).
    #
    # Default depends on the client.
    # Templates: allowed
    url_template: "https://github.mycompany.com/foo/bar/releases/download/{{ .Tag }}/{{ .ArtifactName }}"

    # Git author used to commit to the repository.
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com

    # The project name and current git tag are used in the format string.
    #
    # Templates: allowed
    commit_msg_template: "{{ .ProjectName }}: {{ .Tag }}"

    # Path for the file inside the repository.
    #
    # Default: pkgs/<name>/default.nix
    path: pkgs/foo.nix

    # Your app's homepage.
    homepage: "https://example.com/"

    # Your app's description.
    #
    # Templates: allowed
    description: "Software to create fast and easy drum rolls."

    # License name.
    license: "mit"

    # Setting this will prevent goreleaser to actually try to commit the updated
    # package - instead, it will be stored on the dist folder only,
    # leaving the responsibility of publishing it to the user.
    #
    # If set to auto, the release will not be uploaded to the repository
    # in case there is an indicator for prerelease in the tag e.g. v1.0.0-rc1
    #
    # Templates: allowed
    skip_upload: true

    # Custom install script.
    #
    # Default: 'mkdir -p $out/bin; cp -vr $binary $out/bin/$binary'
    # Templates: allowed
    install: |
      mkdir -p $out/bin
      cp -vr ./foo $out/bin/foo
      installManPage ./manpages/foo.1.gz

    # Custom post_install script.
    # Could be used to do any additional work after the "install" script
    #
    # Templates: allowed
    post_install: |
      installShellCompletion ./completions/*
```

!!! tip
    Learn more about the [name template engine](/customization/templates/).

## `nix-prefetch-url`

The `nix-prefetch-url` binary must be available in the `$PATH` for the
publishing to work.

## Compile from source

GoReleaser does not (yet) generate packages that compile the Go module from
source.
It's planned for next releases, though.

## GitHub Actions

To publish a package from one repository to another using GitHub Actions, you
cannot use the default action token.
You must use a separate token with content write privileges for the tap
repository.
You can check the
[resource not accessible by integration](/errors/resource-not-accessible-by-integration/)
for more information.


