---
title: Global Hooks
---

Some release cycles may need run something before or after everything else.

GoReleaser allows this with the global hooks feature.

=== "OSS"
    The `before` section allows for global hooks that will be executed before the release is started.

    The configuration is straightforward, here is an example will all possible options:

    ```yaml
    # .goreleaser.yml
    before:
      # Templates for the commands to be ran.
      hooks:
      - make clean
      - go generate ./...
      - go mod tidy
      - touch {{ .Env.FILE_TO_TOUCH }}
    ```

=== "PRO"
    !!! success "GoReleaser Pro"
        Global after hooks and the additional options in before hooks (`dir` and `env`) are [GoReleaser Pro features](/pro/).

    The `before` section allows for global hooks that will be executed before the release is started.
    Likewise, the `after` section allows for global hooks that will be executed before the release is started.

    The configuration is straightforward, here is an example will all possible options:

    ```yaml
    # .goreleaser.yml
    # global before hooks
    before:
      # Templates for the commands to be ran.
      hooks:
      - make clean # simple string
      - cmd: go generate ./... # specify cmd
      - cmd: go mod tidy
        dir: ./submodule # specify command working directory
      - cmd: touch {{ .Env.FILE_TO_TOUCH }}
        env:
          FILE_TO_TOUCH: 'something-{{ .ProjectName }}' # specify hook level environment variables

    # global after hooks
    after:
      # Templates for the commands to be ran.
      hooks:
      - make clean
      - cmd: cat *.yaml
        dir: ./submodule
      - cmd: touch {{ .Env.RELEASE_DONE }}
        env:
          RELEASE_DONE: 'something-{{ .ProjectName }}' # specify hook level environment variables
    ```


Note that if any of the hooks fails the release process is aborted.

## Complex commands

If you need to do anything more complex, it is recommended to create a shell script and call it instead.
You can also go crazy with `sh -c "my commands"`, but it gets ugly really fast.

!!! tip
    Learn more about the [name template engine](/customization/templates/).
