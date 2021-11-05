# Azure Pipelines

GoReleaser can also be used within our official [GoReleaser Extensions for Azure DevOps][goreleaser-extension]
through [Visual Studio marketplace][marketplace].

### Task definition

````yaml
- task: goreleaser@0
  inputs:
    version: 'latest'
    distribution: 'goreleaser'
    args: ''
    workdir: '$(Build.SourcesDirectory)'
````

### Task inputs

Following inputs can be used

| Name             | Type    | Default      | Description                                                      |
|------------------|---------|--------------|------------------------------------------------------------------|
| `distribution`   | String  | `goreleaser` | GoReleaser distribution, either `goreleaser` or `goreleaser-pro` |
| `version`**¹**   | String  | `latest`     | GoReleaser version                                               |
| `args`           | String  |              | Arguments to pass to GoReleaser                                  |
| `workdir`        | String  | `$(Build.SourcesDirectory)`          | Working directory (below repository root)                        |
| `install-only`   | Bool    | `false`      | Just install GoReleaser                                          |

> **¹** Can be a fixed version like `v0.132.0` or a max satisfying semver one like `~> 0.132`. In this case this will return `v0.132.1`.
> For the `pro` version, add `-pro` to the string

### Task environment variables

```yaml
...
variables:
- name: GORELEASER_KEY
  value: xxx
...

or short:

...
variables:
  GORELEASER_KEY: xxx
...
```

Following environment variables can be used, as environment variable.

| Name             | Description                           |
|------------------|---------------------------------------|
| `GITHUB_TOKEN`   | [GITHUB_TOKEN](https://help.github.com/en/actions/configuring-and-managing-workflows/authenticating-with-the-github_token) for e.g. `brew` or `gofish` |
| `GORELEASER_KEY` | Your [GoReleaser Pro](https://goreleaser.com/pro) License Key, in case you are using the `goreleaser-pro` distribution                              |

### Example pipeline

Generally there are two ways to define an [Azure Pipeline](https://azure.microsoft.com/en-us/services/devops/pipelines/):
Classic pipelines defined in the UI or YAML pipelines.

Here is how to do it with YAML:

```yaml
# customize trigger to your needs
trigger:
  branches:
    include:
      - main
      - refs/tags/*

variables:
  GO_VERSION: "1.17"

pool:
  vmImage: ubuntu-latest

jobs:
  - job: Test
    steps:
      - task: GoTool@0
        inputs:
          version: "$(GO_VERSION)"
        displayName: Install Go

      - bash: go test ./...
        displayName: Run Go Tests

  - job: Release
    # only runs if Test was successful
    dependsOn: Test
    # only runs if pipeline was triggered from a branch.
    condition: and(succeeded(), startsWith(variables['Build.SourceBranch'], 'refs/tags'))
    steps:
      - task: GoTool@0
        inputs:
          version: "$(GO_VERSION)"
        displayName: Install Go

      - task: goreleaser@0
        inputs:
          version: 'latest'
          distribution: 'goreleaser'
          args: ''
          workdir: '$(Build.SourcesDirectory)'
```

In this example a `Test` job is used to run `go test ./...` to first make sure that there're no failing tests. Only if
that job succeeds and the pipeline was triggered from a tag (because of the defined `condition`) Goreleaser will be run.

[goreleaser-extension]: https://marketplace.visualstudio.com/items?itemName=GoReleaser.goreleaser
[marketplace]: https://marketplace.visualstudio.com/azuredevops
