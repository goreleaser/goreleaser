---
title: Global Hooks
---

Some builds may need pre-build steps before building, e.g. `go generate`.
The `before` section allows for global hooks which will be executed before
the build is started.

The configuration is very simple, here is a complete example:

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

If any of the hooks fails the build process is aborted.

## Complex commands

If you need to do anything more complex, it is recommended to create a shell script and call it instead.
You can also go crazy with `sh -c "my commands"`, but it gets ugly real fast.


## Pro Features

With [GoReleaser Pro](/pro/), things are a bit more flexible: you can specify the dir, environment variables and also global after hooks.

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

!!! info
    Global after hooks is a [GoReleaser Pro feature](/pro/).

!!! tip
    Learn more about the [name template engine](/customization/templates/).
