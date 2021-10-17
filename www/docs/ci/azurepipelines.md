# Azure Pipelines

Generally there're two ways to define an [Azure Pipeline](https://azure.microsoft.com/en-us/services/devops/pipelines/):
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

      - bash: curl -sL https://git.io/goreleaser | bash
        displayName: Run Goreleaser
```

In this example a `Test` job is used to run `go test ./...` to first make sure that there're no failing tests.
Only if that job succeeds and the pipeline was triggered from a tag (because of the defined `condition`) Goreleaser will be run.
