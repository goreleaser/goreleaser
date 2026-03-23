---
title: "Global Hooks"
weight: 90
---

Some release cycles may need to run something before or after everything else.

GoReleaser allows this with the global hooks feature.

{{< tabs >}}
{{< tab "OSS" >}}

The `before` section allows for global hooks that will be executed
**before** the release is started.

The configuration is straightforward, here is an example will all possible
options:

```yaml {filename=".goreleaser.yaml"}
before:
  # Templates for the commands to be ran.
  hooks:
    - make clean
    - go generate ./...
    - go mod tidy
    - touch {{ .Env.FILE_TO_TOUCH }}
```

{{< /tab >}}
{{< tab "Pro" >}}

{{< featpro >}}

The `before` section allows for global hooks that will be executed
**before** the release is started. Likewise, the `after` section allows for
global hooks that will be executed **after** the release is started.

The configuration is straightforward, here is an example will all possible
options:

```yaml {filename=".goreleaser.yaml"}
# global before hooks
before:
  # Commands to be ran.
  #
  # Templates: allowed.
  hooks:
    - make clean # simple string
    - cmd: "go generate ./..." # specify cmd
    - cmd: "go mod tidy"
      # Always prints command output.
      output: true
      # Specify directory.
      dir: ./submodule # specify command working directory
    - cmd: "touch {{ .Env.FILE_TO_TOUCH }}"
      # Specify extra environment variables.
      env:
        - "FILE_TO_TOUCH=something-{{ .ProjectName }}" # specify hook level environment variables
    - cmd: "dotnet tool install --global wix"
      # Make the hook optional:
      # {{< inline_version "v2.7" >}}
      if: '{{ eq .Runtime.Goos "windows" }}'

# global after hooks
after:
  # Commands to be ran.
  #
  # Templates: allowed.
  hooks:
    - make clean
    - cmd: cat *.yaml
      dir: ./submodule
    - cmd: touch {{ .Env.RELEASE_DONE }}
      env:
        - "RELEASE_DONE=something-{{ .ProjectName }}" # specify hook level environment variables
    - cmd: "rm -rf ./something"
      if: '{{ eq .Runtime.Goos "linux" }}'
```

{{< /tab >}}
{{< /tabs >}}

Note that if any of the hooks fails the release process is aborted.

## Complex commands

If you need to do anything more complex, it is recommended to create a shell
script and call it instead. You can also go crazy with `sh -c "my commands"`,
but it gets ugly really fast.

{{< templates >}}
