---
date: 2023-01-30
slug: goreleaser-ko
categories:
  - tutorials
authors:
  - developerguy
---

# Fast and Furious Building OCI compatible Container Images with GoReleaser and ko

GoReleaser and [ko][] are popular open-source, well-recognized projects, especially in the containerization and open-source ecosystem for Go applications.
Many people use these projects for their Go applications because they are pretty straightforward and CI-friendly tools that make your releasing artifacts (binary and container image) process super elegant, which also helps you focus more on developing the business logic rather than planning to release software type of works.

<!-- more -->

I’m so glad to announce that we finally [integrated these fantastic projects](/customization/ko)!

> If you are interested in learning more about the development process of that
> feature, here is the [PR](https://github.com/goreleaser/goreleaser/pull/3653/) you can take a look.

As a result, starting from [GoReleaser v1.15](https://github.com/goreleaser/goreleaser/milestone/17), you can build container images by setting configuration options for ko in GoReleaser without having ko installed on your environment.

This post will be a quick walkthrough to guide people about how things work.

Before diving into that, let’s refresh our minds about these projects with a quick recap.

GoReleaser is a tool for creating and releasing Go projects. It automates the
process of building, packaging, and publishing Go binaries and container images,
basically the [fanciest way of releasing Go projects](https://medium.com/trendyol-tech/the-fanciest-way-of-releasing-go-binaries-with-goreleaser-dbbd3d44c7fb).
It is a super user-friendly, easy-to-use, go-to CLI tool and also provides
[GitHub Actions](https://github.com/goreleaser/goreleaser-action) to be
CI-friendly. It also includes a bunch of features for mitigating the risks of
the software supply chain attacks, such as [generating
SBOMs](/customization/sbom), [signing the artifacts](/customization/sign), and
many others.
To get more detail, here is the [blog post](/blog/supply-chain-security) for
you.

On the other hand, [ko][] is specifically designed for building and publishing
container images for Go projects. But the utmost vital features in ko are that
it doesn’t require you to run any Docker daemon or write well-designed
Dockerfiles to make the build process cache-efficient, fast and secure. The good
news is that ko will consider all these and build OCI-compatible container
images with all the security options enabled by default, [such as using build
arguments while making Go binaries for
reproducibility](https://ko.build/configuration/#overriding-go-build-settings),
[generating SBOMs, and uploading them to the OCI registry](https://ko.build/features/sboms/),
using the
[smallest and CVE-less base
image](https://github.com/ko-build/ko/blob/453bf803e379696a0b9142c772402ba4599cff34/pkg/commands/options/build.go#L35)
from [Chainguard images](https://github.com/chainguard-images/images) and
providing base information using OCI [base image
annotations](https://github.com/opencontainers/image-spec/blob/main/annotations.md)
and also it [makes easier multi-platform
builds](https://ko.build/features/multi-platform/) by using the
cross-compilation in Go. To get more detail, here is the [blog post](https://blog.kubesimplify.com/getting-started-with-ko-a-fast-container-image-builder-for-your-go-applications) for you.

It is worth mentioning that [ko applied to become a CNCF sandbox project the last year](https://opensource.googleblog.com/2022/10/ko-applies-to-become-a-cncf-sandbox-project.html),
and glad to see that this application [got accepted by the CNCF](https://lists.cncf.io/g/cncf-toc/message/7743),
which means that ko is now officially a CNCF Sandbox project.

Without further ado, let’s dive into the details of the integration by showing it in a real-world example.

> You will find all the source code in GitHub repository,
> [here](https://github.com/developer-guy/goreleaser-with-ko).

Let’s start with creating a proper directory to host the source code:

```bash
$ mkdir -p goreleaser-with-ko
$ cd goreleaser-with-ko
```

Next, initialize the project:

```bash
go mod init github.com/<username>/goreleaser-with-ko
cat <<EOF > main.go
package main

import (
  "fmt"
  "os"
)
var (
  // Version is the current version of the application.
  Version = "main"
)
func main() {
  fmt.Fprintf(os.Stdout, "GoReleaser supports ko! Version: %s", Version)
}
EOF
```

It is time to create the configuration file for GoReleaser, which is
[.goreleaser.yml](/customization).

The easiest way of creating that file is run: `goreleaser init`, which requires
GoReleaser CLI to be installed on your environment; please refer to the
installation page [here](/install) to install it.

```bash
# it will create the .goreleaser.yml configuration file
# with bunch of default configuration options.
$ goreleaser init
• Generating .goreleaser.yaml file
• config created; please edit accordingly to your needs file=.goreleaser.yaml
```

Next, set ko configuration options into .goreleaser.yml. Fortunately, we have
good documentation explaining how we can do this [here](/customization/ko).

```bash
$ cat <<EOF >> .goreleaser.yml
kos:
  - id: goreleaser-with-ko
    platforms:
    - linux/amd64
    - linux/arm64
    tags:
    - latest
    - '{{.Tag}}'
    bare: true
    flags:
    - -trimpath
    ldflags:
    - -s -w
    - -extldflags "-static"
    - -X main.Version={{.Tag}}
EOF
```

Finally, we’ll automate this workflow on the GitHub Actions platform.
To do this, we need to create a proper folder structure, `.github/workflows` and
put the workflow file into it:

```bash
$ mkdir -p .github/workflows
$ cat <<EOF > .github/workflows/release.yaml
name: Releasing artifacts with GoReleaser and ko
on:
  push:
    tags:
      - 'v*'
permissions:
   contents: write # needed to write releases
   packages: write # needed for ghcr access
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
        with:
          fetch-depth: 0 # this is important, otherwise it won't checkout the full tree (i.e. no previous tags)
      - uses: actions/setup-go@v3
        with:
          go-version: 1.19
          cache: true
      - uses: goreleaser/goreleaser-action@v4 # run goreleaser
        with:
          version: latest
          args: release --rm-dist
        env:
          KO_DOCKER_REPO: ghcr.io/${{ github.repository_owner }}/goreleaser-with-ko
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
EOF
```

As you saw from the file above, we didn’t do anything special about ko
installation, but in case you need to install it into your workflow, you can use
the [setup-ko](https://github.com/ko-build/setup-ko) GitHub Action for that. But how?

Since ko’s core packages that provide such building and publishing capabilities
are exported functions, you can use them in your own Go projects to get more
detail [here](https://ko.build/advanced/go-packages/).
The following projects are great examples of that:

- [terraform-provider-ko](https://github.com/ko-build/terraform-provider-ko/blob/main/internal/provider/resource_ko_image.go)
- [miniko](https://github.com/imjasonh/miniko)
- And now, [GoReleaser](https://github.com/goreleaser/goreleaser/blob/main/internal/pipe/ko/ko.go)

And that’s it. All you need to do at that point is give a tag to your project and wait for the GitHub workflow to be completed to release your software.

```bash
$ git commit -m"initial commit" -s
$ git tag v0.1.0 -m"first release"
$ git push origin v0.1.0
```

One last note: please remember to use this feature and provide feedback to help us improve this process. Thanks for reading; I hope you enjoyed it; see you in the next blog posts.

[ko]: https://ko.build/
