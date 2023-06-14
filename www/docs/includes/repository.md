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
      # Templates: allowed
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"

      # Sets up pull request creation instead of just pushing to the given branch.
      # Make sure the 'branch' property is different from base before enabling
      # it.
      #
      # Since: v1.17
      pull_request:
        # Whether to enable it or not.
        enabled: true

        # Whether to open the PR as a draft or not.
        #
        # Default: false
        # Since: v1.19
        draft: true

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
      #
      # Since: v1.18
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
