---
date: 2023-01-10
slug: azure-devops
categories:
  - tutorials
authors:
  - dirien
---

# Releasing multi-platform container images with GoReleaser in Azure DevOps

<!-- more -->

## Introduction

In this article, we learn how to use [GoReleaser](https://goreleaser.com/) to build and release a multi-platform container image to [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/) in [Azure DevOps](https://azure.microsoft.com/en-us/services/devops/).

This is particularly interesting for teams, who are using mainly Azure and Azure DevOps for their projects and want to build and release container images to Azure Container Registry.

I try to follow the great article on how to create multi-platform container
images using GitHub actions written by Carlos, the core maintainer of
GoReleaser. If you had no chance to read his blog, [here](https://carlosbecker.com/posts/multi-platform-docker-images-goreleaser-gh-actions/) is the link to it.

Before we start, let’s take a look on the prerequisites.

## Prerequisites

- [Azure DevOps](https://azure.microsoft.com/en-us/services/devops/) account.
- [The GoReleaser Azure DevOps Extension](https://marketplace.visualstudio.com/items?itemName=GoReleaser.goreleaser) installed.
- [Azure](https://azure.microsoft.com/en-us/) account.
- [GoReleaser](https://goreleaser.com/install/) installed on your local machine.

## The sample application

Before we can start to set up our pipeline and infrastructure components, lets have a look at the sample application we are going to use in this demo. To keep things simple, I created basic Hello World server using mux library from the Gorilla Web Toolkit.

Add the library to the `go.mod` file:

```gomod
module dev.azure.com/goreleaser-container-example

go 1.19

require github.com/gorilla/mux v1.8.0
```

After adding the library, we can move over to implement the basic logic of the application. The server should return a Hello World! string, when we curl the root path of the server.

In mux this is done, with adding a new route to the router and adding a handler function to it. In my case called HelloWorldHandler.

Then we can start the server and listen on port 8080.

```go
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

const (
	// Port is the port the server will listen on
	Port = "8080"
)

func HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World!"))
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/", HelloWorldHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = Port
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "TEST") {
			log.Printf("%s", env)
		}
	}
	log.Println("Listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
```

As we want to create a container image, we need to add a Dockerfile. GoReleaser will then build our container image by copying the previously built binary into the container image. Remember: We don't want to rebuild the binary. So no multi-stage Dockerfile. This way we are sure, that the same binary is used for all distribution methods GoReleaser is offering, and we intended to use.

```Dockerfile
# Dockerfile
FROM cgr.dev/chainguard/static@sha256:bddbb08d380457157e9b12b8d0985c45ac1072a1f901783a4b7c852e494967d8
COPY goreleaser-container-example \
    /usr/bin/goreleaser-container-example
ENTRYPOINT ["/usr/bin/goreleaser-container-example"]
```

![Chainguard logo](https://cdn-images-1.medium.com/max/2000/0*3X76j809VWDnCLxY)

You may spot that I use a static container image as base image from Chainguard. Chainguard images are designed for minimalism and security in mind. Many of the images provided by Chainguard are distroless, which means they do not contain a package manager or any other programs that are not required for the specific purpose of the image. Chainguard images are also scanned for vulnerabilities and are regularly updated. You can find more information about Chainguard images here:
[**Chainguard Images**
*Chainguard Images are security-first container base images that are secure by default, signed by Sigstore, and include…*www.chainguard.dev](https://www.chainguard.dev/chainguard-images)

Let’s pause a minute here and test that everything is working as expected. We can test the application by running it locally:

```bash
GOOS=linux GOARCH=amd64 go build -o goreleaser-container-example .
docker buildx build --platform linux/amd64 -t goreleaser-container-example .
docker run -p 8080:8080 goreleaser-container-example
```

After spinning up the container, you should see the following output:

```bash
2023/01/10 10:49:31 Listening on port 8080
```

And if we run a curl command in another terminal, we should see the following output:

```bash
curl localhost:8080
Hello World!
```

Perfect! Everything works as we expected it. Now we can start working on the GoReleaser parts.

## GoReleaser config file

We need to provide a goreleaser.yaml config file in the root of our project to tell GoReleaser what to do during the release process. In our case to let GoReleaser to build our container image. To create the goreleaser.yaml we can run following command:

```bash
goreleaser init
```

This should generate the config file for us:

```bash
  • Generating .goreleaser.yaml file
  • config created; please edit accordingly to your needs file=.goreleaser.yaml
```

The good part when using the init command is, that the goreleaser.yaml comes with some default values. We need to change content as we not need everything GoReleaser is doing by default. Here is the content of the goreleaser.yaml for this demo:

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
    goos:
      - linux
      - darwin
```

Later we add the part needed to create the multi-platform container images but for now we dry-run the release process with following GoReleaser command:

```bash
goreleaser release --snapshot --rm-dist
```

Next to the logs of GoReleaser release process, you should also have a dist folder with all the binaries in it.

> _Exclude this folder in your .gitignore file, to prevent accidentally committing the binaries to your repository._

## Azure Container Registry

> _If you already have an Azure Container Registry you can skip the parts of the creation of the Azure Container Registry._

There are several ways, you can create a container registry: You can use the Azure Portal, the Azure CLI, the Azure PowerShell or your favorite Infrastructure as Code tool of choice.

In this demo, I will use the Azure CLI to create the container registry. You can
find more information about the Azure CLI
[here](https://docs.microsoft.com/en-us/cli/azure/?view=azure-cli-latest).

First log into the Azure account with the Azure CLI:

```bash
az login
```

We then need to create the resource group and then the container registry service with following commands:

```bash
# create a resource group in WestEurope datacenter
az group create --name goreleaser-rg --location westeurope
# create the Azure Container registry
az acr create --resource-group goreleaser-rg --name mygoreleaserregistry --sku Basic
```

When the container registry is up and running, we can add the dockers configuration to our goreleaser.yaml. But we need to name of our container registry beforehand.

Use following command to retrieve the name:

```bash
az acr show --resource-group goreleaser-rg --name mygoreleaserregistry --query loginServer --output tsv
```

This is the new part we need to add to our `goreleaser.yaml,` to activate the build of the container image and manifest.

If you want to know more about the manifest files, I wrote an article about it
[here](/blog/docker-manifests).

```yaml
---
dockers:
  - image_templates:
      [
        "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-amd64",
      ]
    goarch: amd64
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/amd64
  - image_templates:
      [
        "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-arm64",
      ]
    goarch: arm64
    dockerfile: Dockerfile
    use: buildx
    build_flag_templates:
      - --platform=linux/arm64/v8
docker_manifests:
  - name_template: "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}"
    image_templates:
      - "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-amd64"
      - "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-arm64"
  - name_template: "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:latest"
    image_templates:
      - "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-amd64"
      - "mygoreleaserregistry.azurecr.io/{{ .ProjectName }}:{{ .Version }}-arm64"
```

## Azure DevOps

With the infrastructure done and GoReleaser config finished, we can set up Azure DevOps Service.

![Switch to the Service Connections screen](https://cdn-images-1.medium.com/max/4060/1*4n5ZoSZ6HMNp0kL--zYFPA.png)_Switch to the Service Connections screen_

![Click on the New service connection button](https://cdn-images-1.medium.com/max/4060/1*WBYOMjBmxl_LAEhnUcBSDg.png)_Click on the New service connection button_

![Select Azure Container Registry and connect your Azure subscription to it](https://cdn-images-1.medium.com/max/4060/1*kGcBwOXxuKwQsnkwo2870A.png)_Select Azure Container Registry and connect your Azure subscription to it_

Time for the last part of our demo: Setting up the Azure DevOps pipeline. I will not go too much into detail about the pipeline, as this is not the focus of this demo. But I will show you the important parts of the pipeline.

First notable part is the multi-platform command task. I simply followed the instructions from this [article](https://learn.microsoft.com/en-us/azure/devops/pipelines/ecosystems/containers/build-image?view=azure-devops#how-do-i-build-linux-container-images-for-architectures-other-than-x64) on how to setup the task to build multiarch images.

Next section in the pipeline is the GoReleaser task. This task is using the
GoReleaser extension from the Azure DevOps Marketplace. You can find more
information about the extension [here](https://marketplace.visualstudio.com/items?itemName=ms-azuretools.goreleaser-task).

I just added the args field and set the value to release `--rm-dist` and defined a condition to only run the task on a tag as GoReleaser will not release on a "dirty" git state.

This is the complete pipeline:

```yaml
trigger:
  branches:
    include:
      - main
      - refs/tags/*
variables:
  GO_VERSION: "1.19.4"
  DOCKER_BUILDKIT: 1
pool:
  vmImage: ubuntu-latest
jobs:
  - job: Release
    steps:
      - task: GoTool@0
        inputs:
          version: "$(GO_VERSION)"
        displayName: Install Go
      - task: Docker@2
        inputs:
          containerRegistry: "goreleaser"
          command: "login"
          addPipelineData: false
          addBaseImageData: false
      - task: CmdLine@2
        displayName: "Install multiarch/qemu-user-static"
        inputs:
          script: |
            docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
      - task: goreleaser@0
        condition: and(succeeded(), startsWith(variables['Build.SourceBranch'], 'refs/tags/'))
        inputs:
          version: "latest"
          distribution: "goreleaser"
          workdir: "$(Build.SourcesDirectory)"
          args: "release --rm-dist"
```

To run a release, you need to create a tag in Azure to get the release process started.

![Logs produced during the release process](https://cdn-images-1.medium.com/max/4060/1*rawcazzmdDWXUzeo-YAIlQ.png)_Logs produced during the release process_

And you should see in the Repository tab of your Azure Container Registry service in the Azure Portal UI the multi-platform container images.

![List of all produced multi-platform container images](https://cdn-images-1.medium.com/max/7184/1*Z8mRJwHIv3o9jWlubU_hhQ.png)_List of all produced multi-platform container images_

## Conclusion

In this demo, I showed you how to create a multi-platform container image using GoReleaser and Azure DevOps and store this image in Azure Container Registry for further usage in your Container based Azure services.

Setting up all the parts was pretty straight forward where GoReleaser is doing the heavy lifting for us.

Go ahead and give it a try and let me know what you think about it.
