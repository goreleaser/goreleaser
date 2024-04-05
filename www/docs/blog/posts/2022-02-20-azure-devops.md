---
date: 2022-02-20
slug: azure-devops
categories:
  - tutorials
authors:
  - dirien
---

# Use GoReleaser With Azure DevOps

In this blog article, I want to show how to use **GoReleaser** in **Azure DevOps**.

<!-- more -->

![](https://cdn-images-1.medium.com/max/2000/1*hAK9NMa-YbSnTQdSEoX7Gw.png)

In this blog article, I want to show how to use **GoReleaser** in **Azure
DevOps**.

### But why? Are not everyone using GitHub?

Exactly, not everyone is using GitHub. Actually, there are many companies who
use the Azure Cloud with **Azure DevOps**.

## What is Azure DevOps?

**Azure DevOps** provides developer services for allowing teams to plan work,
collaborate on code development, and build and deploy applications.

**Azure DevOps** provides integrated features that you can access through your
web browser or IDE client. You can use one or more of the following standalone
services based on your business needs:

- **Azure Repos** provides Git repositories.
- **Azure Pipelines** provides build and release services to support continuous
  integration and delivery of your applications.
- **Azure Boards** delivers a suite of Agile tools to support planning and
  tracking work, code defects, and issues using Kanban and Scrum methods.
- **Azure Test Plans** provides several tools to test your apps, including
  manual/exploratory testing and continuous testing.
- **Azure Artifacts** allows teams to share packages such as Maven, npm, NuGet,
  and more from public and private sources and integrate package sharing into your
  pipelines.

## Install GoReleaser Via The Marketplace

![](https://cdn-images-1.medium.com/max/4144/1*NCY5i6iCEPW5ZpNhYeB6Jg.png)

**GoReleaser** offers a [Plugin via the
Marketplace](https://marketplace.visualstudio.com/items?itemName=GoReleaser.goreleaser).
The installation itself is done via some clicks in the UI and you are ready to
start!

In your pipeline editor you can lookup the task:

![](https://cdn-images-1.medium.com/max/2000/1*U-zVyao5qwgjfzTcGHgSJA.png)

And quickly change the default settings to fit with your use case!
For example set a specific version or execute certain **GoReleaser** commands.

See the official docs for more details
[https://github.com/goreleaser/goreleaser-azure-devops-extension](https://github.com/goreleaser/goreleaser-azure-devops-extension)

![](https://cdn-images-1.medium.com/max/2000/1*Aqtx-0KvADotNmeX7X1HTQ.png)

## Finally The Demo!

```go
package main

import (
	"fmt"
)

var (
	version = "0.0.1"
	commit  = "none"
	date    = "none"
	builtBy = "none"
)

func main() {
	fmt.Println("Version:\t", version)
	fmt.Println("Commit:\t\t", commit)
	fmt.Println("Date:\t\t", date)
	fmt.Println("Built by:\t", builtBy)
}
```

Before we head over to the configure the pipeline, let us create the
`.goreleaser.yaml`

```yaml
# This is an example .goreleaser.yml file with some sensible defaults.
# Make sure to check the documentation at https://goreleaser.com
before:
  hooks:
    # You may remove this if you don't use go modules.
    - go mod tidy
builds:
  - env:
      - CGO_ENABLED=0
    goarch:
      - amd64
      - arm64
    goos:
      - linux
      - darwin

project_name: goreleaser-ado

checksum:
  name_template: "checksums.txt"

snapshot:
  name_template: "{{ incpatch .Version }}-next"

source:
  enabled: true

release:
  disable: true

sboms:
  - artifacts: archive
  - id: source
    artifacts: source

signs:
  - cmd: cosign
    certificate: "${artifact}.pem"
    args:
      - sign-blob
      - "-key=cosign.key"
      - "--output-certificate=${certificate}"
      - "--output-signature=${signature}"
      - "${artifact}"
    artifacts: checksum
    output: true
    stdin: "{{ .Env.COSIGN_PASSWORD }}"

docker_signs:
  - cmd: cosign
    artifacts: images
    output: true
    args:
      - "sign"
      - "-key=cosign.key"
      - "${artifact}"
    stdin: "{{ .Env.COSIGN_PASSWORD }}"

dockers:
  - image_templates: ["dirien/{{ .ProjectName }}:{{ .Version }}-amd64"]
    goarch: amd64
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/amd64
  - image_templates: ["dirien/{{ .ProjectName }}:{{ .Version }}-arm64"]
    goarch: arm64
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/arm64/v8

docker_manifests:
  - name_template: "dirien/{{ .ProjectName }}:{{ .Version }}"
    image_templates:
      - "dirien/{{ .ProjectName }}:{{ .Version }}-amd64"
      - "dirien/{{ .ProjectName }}:{{ .Version }}-arm64"
  - name_template: "dirien/{{ .ProjectName }}:latest"
    image_templates:
      - "dirien/{{ .ProjectName }}:{{ .Version }}-amd64"
      - "dirien/{{ .ProjectName }}:{{ .Version }}-arm64"
```

Here, we going to create **linux** and **darwin** binary, the corresponding
container, create the SBoM with syft and sign everything via cosign.

> Here is one first important steps: you need to disable the **release** step in
> GoReleaser.
> Azure DevOps does not work the same way as GitHub what releases concerns.
> We handle the upload of the artifacts differently.

If you need more infos, for the different settings and possibilities inside
**GoReleaser**, head over to the official documentation
[https://goreleaser.com/intro/](https://goreleaser.com/intro/)

### Service

Connection in Azure DevOps As we going to upload the image to Docker Hub, we
need to create in **Azure Devops** the Service Connection.

Go to **Project Settings** and click **Service connections**:
![](https://cdn-images-1.medium.com/max/2000/1*QDD9dVR5ZYUtm4qh5Izy2w.png)
![](https://cdn-images-1.medium.com/max/2000/1*PMGoYUHoda1bVumVGW1Fow.png)

Choose **Docker Registry**:
![](https://cdn-images-1.medium.com/max/2000/1*JmkHy2p5pjhc7llWCHjChw.png)

In the detail view, select **Docker Hub **and then enter your details, like
**Docker ID**, **Docker Password** and the **Service Connection Name**:
![](https://cdn-images-1.medium.com/max/2000/1*WCT6xaaNtoT6Y-Ws7-0qOQ.png)

Click **Verify and save**, we will come back to the service connection in our
Pipeline code.

### The Azure Pipeline File

Now starts the fun part, the creation of the actual Azure Pipeline.
If you are completely new to **Azure DevOps** pipeline, I highly suggest to
checkout the
[docs](https://docs.microsoft.com/en-us/azure/devops/pipelines/create-first-pipeline?view=azure-devops&tabs=java%2Ctfs-2018-2%2Cbrowser)
from Microsoft.

In our example, we going to write the pipeline only as code (there is a
deprecated UI only option too! But meh!).

Azure Pipeline are written in **yaml.**

```yaml
# Starter pipeline
# Start with a minimal pipeline that you can customize to build and deploy your code.
# Add steps that build, run tests, deploy, and more:
# https://aka.ms/yaml

pr:
  branches:
    include:
      - main

trigger:
  tags:
    include:
      - "*"
  branches:
    include:
      - "*"

jobs:
  - job: build
    pool:
      vmImage: ubuntu-latest
    steps:
      - task: GoTool@0
        displayName: "Install Go"
        inputs:
          version: "1.17"
      - task: CmdLine@2
        displayName: "Build and Test"
        inputs:
          script: |
            go mod tidy
            go build .
  - job: release
    dependsOn: build
    displayName: Release via GoReleaser
    condition: and(succeeded(), startsWith(variables['Build.SourceBranch'], 'refs/tags/'))
    variables:
      - group: cosign
    pool:
      vmImage: ubuntu-latest
    steps:
      - task: GoTool@0
        displayName: "Install Go"
        inputs:
          version: "1.17"
      - task: CmdLine@2
        displayName: "Install Syft"
        inputs:
          script: |
            curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
      - task: CmdLine@2
        displayName: "Install cosign"
        inputs:
          script: |
            curl -sLO https://github.com/sigstore/cosign/releases/download/v1.5.2/cosign-linux-amd64
            chmod +x cosign-linux-amd64
            mv cosign-linux-amd64 /usr/local/bin/cosign
      - task: Docker@2
        inputs:
          containerRegistry: "dirien-docker-hub"
          command: "login"
          addPipelineData: false
          addBaseImageData: false
      - task: goreleaser@0
        inputs:
          version: "latest"
          distribution: "goreleaser"
          args: "release --rm-dist"

      - task: CopyFiles@2
        displayName: "Copy GoReleaser dist folder to ArtifactStagingDirectory"
        inputs:
          Contents: |
            dist/*.tar.gz
            dist/*.zip
            dist/*.txt
            dist/*.sbom
            dist/*.sig
            cosign.pub
          TargetFolder: "$(Build.ArtifactStagingDirectory)"
          CleanTargetFolder: true
          OverWrite: true
          flattenFolders: true

      - task: PublishBuildArtifacts@1
        displayName: "Publish GoReleaser release artifacts"
        inputs:
          ArtifactName: "GoReleaser release"
```

The pipeline consist of two different jobs parts:

- the **build** job, run every time something changes on any branch or when a
  pull request gets created. Here we can run our tests, linting, SAST to get
  quickly feedback.
- the **release** job, will run only when a git tag gets created (see the
  condition tag under the job tag). Creating a git tag is part of the release
  process. Similar as we do in the GitHub Flow.

During the release job, we download [Anchore
syft](https://github.com/anchore/syft) and
[cosign](https://github.com/sigstore/cosign) as we going to need them during the
**goreleaser** task.
Currently there is no native task for this in **Azure DevOps**. We just use the
**CmdLine** task and curl the binaries.

![](https://cdn-images-1.medium.com/max/2496/1*O01UvqQCv8255Kj9JEvxiw.png)

It is also important to log into your **Docker Hub** account, via the **Service
Connection** we created earlier.
The **Docker** task takes care of the actual login.

![](https://cdn-images-1.medium.com/max/2000/1*_uMeZymqEJ0Xqq64OP_UXA.png)

Now we can call our **GoReleaser** task.

![](https://cdn-images-1.medium.com/max/2000/1*blEbdGwd8PXZ9ExCScfYtw.png)

### Azure Pipeline Secret Library

For cosign, I use a password, I stored in the Azure Pipeline Library as secret
variable.

![](https://cdn-images-1.medium.com/max/4028/1*ioSS8X6yT4qFNQrCrQbAYA.png)

In my pipeline code, I will pass this value as environment variable via the
variables tag.

![](https://cdn-images-1.medium.com/max/2000/1*CTox6hgrOCgaTs9AFJWUhQ.png)

In this demo, I am going to publish the release artifacts as build artifacts.

![](https://cdn-images-1.medium.com/max/2244/1*nwH4Ej9GbQ6RdE_MfYMelw.png)

The task **CopyFiles** collects some files from the **dist** folder and the
cosign public key and **PublishBuildArtifacts** publish them.
You will find the artifacts on the pipeline detail

![](https://cdn-images-1.medium.com/max/2000/1*GWCiLGBnHOnCjyRuqrPEEw.png)

![](https://cdn-images-1.medium.com/max/4220/1*MLeku1FpKCFMv7bfNvu_eA.png)

Of course, you can use other targets too, like a cloud native storage.

You can check out the
[How to use GoReleaser with Cloud Native Storage](/blog/cloud-native-storage)
post for more details on this subject

### Release the kraken äh app!

Head over to **tags** menu and create a new tag in the UI

![](https://cdn-images-1.medium.com/max/2000/1*zIYaYhDk1rSpd5_si5mHIA.png)

![](https://cdn-images-1.medium.com/max/4472/1*Zaq4EUjn3egC5_3VLVQA5Q.png)

Your pipeline should immediately start to run:

![](https://cdn-images-1.medium.com/max/5528/1*h5YYGAPjIXrcympI01H9yA.png)

And both jobs should run:

![](https://cdn-images-1.medium.com/max/5592/1*TawbQohuoGdG4sO9xo0YLw.png)

![](https://cdn-images-1.medium.com/max/2000/0*GY5jcM6UErG39iF2.jpg)

And this is pretty much all of it! As I promised, very easy and straight forward
we can implement **GoReleaser** in **Azure DevOps**, similar we would use it in
GitHub

![](https://cdn-images-1.medium.com/max/2560/0*1ZhRSdH-Se88gOdj.jpg)

### Caveat:

- I use in cosign not the _keyless_ approach, as
  I am not sure that it will work for **Azure DevOps**. So I generated a keypair
  and committed the public and private key into the repository.
