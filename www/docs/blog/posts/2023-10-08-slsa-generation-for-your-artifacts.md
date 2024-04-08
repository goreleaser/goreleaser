---
date: 2023-10-11
slug: slsa-generation-for-your-artifacts
categories:
  - tutorials
authors:
  - developerguy
  - dentrax
---

# Stay Calm and SLSA: Generating SLSA Provenance for Your Artifacts with GoReleaser and slsa-github-generator

In an age where software is at the heart of nearly every aspect of our lives, software supply chain security has become paramount. It involves a series of measures and practices aimed at ensuring the reliability and safety of the software we use daily. As cyber threats continue to evolve, the need for robust software supply chain security has never been greater. Organizations must take steps to protect their software development and distribution processes from potential vulnerabilities and attacks.

<!-- more -->

SLSA provenance, short for Supply Chain Levels for Software Artifacts, is an emerging concept that revolutionizes the way we think about software supply chain security. It offers a comprehensive framework to track the lineage and trustworthiness of software components, thereby enhancing overall security.

The core idea behind SLSA provenance is to create a transparent and auditable trail of every software component's journey, from its creation to deployment. This ensures that any tampering or unauthorized changes can be quickly identified and mitigated. Software supply chain security and SLSA provenance are intrinsically linked, as the latter serves as a critical tool to bolster the former.

Together, they provide a robust defense against the growing threats posed by malicious actors in the digital realm. In a world where software vulnerabilities can have far-reaching consequences, the adoption of SLSA provenance is a proactive step toward fortifying our software supply chains and making them more resilient to cyberattacks.

GoReleaser takes the ever-growing risks in the realm of software supply chain security incredibly seriously. From the onset of this era of heightened security concerns, GoReleaser has been at the forefront, continuously adding features to safeguard your artifacts against potential software supply chain attacks such as [generating an SBOMs](https://goreleaser.com/customization/sbom/), [signing your artifacts](https://goreleaser.com/customization/docker_sign/), and more.

> _If you want to learn more about the general software supply chain security features supported by GoReleaser, check out our [blog post](/blog/supply-chain-security/) on the topic._

In this blog post, we will explore how GoReleaser can help you generate SLSA provenance for your artifacts and how you can leverage the slsa-github-generator to automate the process.

## slsa-github-generator

I would like to start with my favorite quote:

> _Each of these attacks could have been prevented if there were a way to detect that the delivered artifacts diverged from the expected origin of the software. But until now, generating verifiable information that described where, when, and how software artifacts were produced (information known as provenance) was difficult. This information allows users to trace artifacts verifiably back to the source and develop risk-based policies around what they consume._ - [Improving software supply chain security with tamper-proof builds](https://security.googleblog.com/2022/04/improving-software-supply-chain.html)

Unfortunately, provenance generation is not widely supported yet but hopefully will be in the future. And this is where the slsa-github-generator comes into play.

Thanks to the SLSA community, they developed a collection of GitHub reusable workflows called [slsa-github-generator](https://github.com/slsa-framework/slsa-github-generator) that can help us with the solving of the problem we mentioned in the quote. It is a powerful tool designed to simplify the process of generating SLSA provenances for your GitHub-hosted projects. It seamlessly integrates with your GitHub repositories, providing an efficient way to enhance the security and trustworthiness of your software supply chain by automatically creating and managing SLSA provenance records.

There are different types of workflows/builders that you can use depending on your needs provided by the slsa-github-generator.

- [Docker](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/docker)
- [Generic](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/generic)
- [Go](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/go)
- [Maven](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/maven)
- [Nodejs](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders/nodejs)

and many [more](https://github.com/slsa-framework/slsa-github-generator/tree/main/internal/builders)...

Without much further ado, let's jump right into the demo and see how we can use the slsa-github-generator to generate SLSA provenance for our artifacts.

> You can find all the code used in this demo in the [goreleaser-example-slsa-provenance](https://github.com/goreleaser/goreleaser-example-slsa-provenance) repository on GitHub.

We said artifacts a lot, but what are artifacts really? In the context of this blog post, artifacts are the binaries and the container images that GoReleaser generates for your project. If you are familiar enough with GoReleaser _-if you are not please refer to the [Quick Start](https://goreleaser.com/quick-start/) guide-_, GoReleaser uses a configuration file called `.goreleaser.yml` to define how to build and release your project. In this file, you can define the artifacts that you want to generate for your project.

Let's take a look at the `.goreleaser.yml` file of our demo project:

```yaml
---
builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin

kos:
  - repository: ghcr.io/goreleaser/goreleaser-example-slsa-provenance
    tags:
      - "{{.Tag}}"
      - "{{ if not .Prerelease }}latest{{ end }}"
    bare: true
    preserve_import_paths: false
    sbom: none
    platforms:
      - all
    flags:
      - -trimpath
    ldflags:
      - -s -w
```

I trimmed the file a bit to make it easier to read. As you can see, we are building our binaries for the `linux`, `windows`, and `darwin` operating systems as we defined in the `builds` section, thanks to the built-in cross-compilation support in Golang. We are also using the [kos](https://goreleaser.com/customization/ko/) integration to build a container image for our project. It is a new way to build container images your project using [ko](https://ko.build). We are also using the `ghcr.io/goreleaser/goreleaser-example-slsa-provenance` repository to push our container image both with the `latest` and the `{{.Tag}}` tag.

> _If you want to learn more about the ko tool, check out our [blog post](https://blog.kubesimplify.com/getting-started-with-ko-a-fast-container-image-builder-for-your-go-applications/)._

Now that we have a better understanding of what artifacts are and the `.goreleaser.yml' is, let's see how we can use the slsa-github-generator to generate SLSA provenance for our artifacts.

Before than that we should talk a little bit about the [GitHub Actions](https://docs.github.com/en/actions/quickstart) platform.

GitHub Actions is an automation and continuous integration/continuous deployment (CI/CD) platform provided by GitHub, which is a widely used web-based platform for version control and collaboration among software developers. GitHub Actions allows you to automate various tasks and workflows in your software development process directly within your GitHub repositories.

To use GitHub Actions, you need to create a workflow file in your repository (_under .github/workflows_). A workflow file is a YAML file that contains a set of instructions that define the steps of your workflow. You can create a workflow file manually or use a workflow template provided by GitHub. GitHub Actions provides a wide range of workflow templates that you can use to automate various tasks and processes in your software development lifecycle. You can also create your own custom workflow templates to suit your specific needs for better reusability and consistency. This is where [reusable workflows](https://docs.github.com/en/actions/using-workflows/reusing-workflows) comes into play.

GitHub Actions and reusable workflows are powerful tools for streamlining your development process, improving code quality, and ensuring consistent practices across your projects. They provide the flexibility and scalability needed to automate tasks and customize your development workflow to meet the specific needs of your software projects.

It's important to understand reusable workflows since the slsa-github-generator is mostly about reusable workflows.

Let's have a first look at the GitHub workflow file that we will use to generate SLSA provenance for our artifacts:

```yaml
---
binary-provenance:
  needs: [goreleaser]
  permissions:
    actions: read # To read the workflow path.
    id-token: write # To sign the provenance.
    contents: write # To add assets to a release.
  uses: slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v1.9.0
  with:
    base64-subjects: "${{ needs.goreleaser.outputs.hashes }}"
    upload-assets: true # upload to a new release

image-provenance:
  needs: [goreleaser]
  permissions:
    actions: read
    id-token: write
    packages: write
  uses: slsa-framework/slsa-github-generator/.github/workflows/generator_container_slsa3.yml@v1.9.0
  with:
    image: ${{ needs.goreleaser.outputs.image }}
    digest: ${{ needs.goreleaser.outputs.digest }}
    registry-username: ${{ github.actor }}
  secrets:
    registry-password: ${{ secrets.GITHUB_TOKEN }} # you should provide registry-password, if you are using private registry like ghcr.io
```

At the time of writing this `1.9.0` is the latest version of the slsa-github-generator. As you can see, we are using two different reusable workflows to generate SLSA provenance for our artifacts. The first one is the `generator_generic_slsa3.yml` workflow, which is used to generate SLSA provenance for our binaries. The second one is the `generator_container_slsa3.yml` workflow, which is used to generate SLSA provenance for our container images.

That's it, we are done. We can now generate SLSA provenance for our artifacts. Confess, you didn't expect this task to be so straightforward. 🤣

Let's continue with explaining them in more detail.

First, as you might have noticed, we are using the `needs` keyword to define the dependencies of our workflow. In this case, we are saying that our workflow depends on the `goreleaser` job. This means that our workflow will run after the `goreleaser` job is completed successfully. This is important because we need the artifacts generated by the `goreleaser` job to generate SLSA provenance for them.

Next, we are using the `permissions` keyword to define the permissions of our job. This is a new feature of GitHub Actions that allows you to define the permissions of your workflow. This is important because we need to define the permissions of our workflow to be able to generate SLSA provenance for our artifacts. In this case, we are saying that our workflow needs the `read` permission to read the workflow path, the `write` permission to sign the provenance, and the `write` permission to add assets to a release.

As we mentioned we have to wait until the `goreleaser` job to be finished before we generate SLSA provenance because we need some output from the `goreleaser` job to generate SLSA provenance.

We are using the `outputs` keyword to define the outputs of our job. In this case, we are saying that our workflow needs the `hashes` output from the `goreleaser` job. This is important because we need the hashes of our artifacts to generate SLSA provenance for them. We are also saying that our workflow needs the `image` and the `digest` output from the `goreleaser` job. This is important because we need the image and the digest of our container image to generate SLSA provenance for them.

Finally, we are using the `with` keyword to define the inputs of our job. In this case, we are saying that our workflow needs the `hashes`, `image.name` and `image.digest` output from the `goreleaser` job.

Let's have a look at the how we generate these outputs in the `goreleaser` job:

```yaml
...
jobs:
  goreleaser:
    outputs:
      hashes: ${{ steps.binary.outputs.hashes }}
      image: ${{ steps.image.outputs.name }}
      digest: ${{ steps.image.outputs.digest }}
      ...

      - name: Generate binary hashes
        id: binary
        env:
          ARTIFACTS: "${{ steps.goreleaser.outputs.artifacts }}"
        run: |
          set -euo pipefail

          checksum_file=$(echo "$ARTIFACTS" | jq -r '.[] | select (.type=="Checksum") | .path')
          echo "hashes=$(cat $checksum_file | base64 -w0)" >> "$GITHUB_OUTPUT"

      - name: Image digest
        id: image
        env:
          ARTIFACTS: "${{ steps.goreleaser.outputs.artifacts }}"
        run: |
          set -euo pipefail
          image_and_digest=$(echo "$ARTIFACTS" | jq -r '.[] | select (.type=="Docker Manifest") | .path')
          image=$(echo "${image_and_digest}" | cut -d'@' -f1 | cut -d':' -f1)
          digest=$(echo "${image_and_digest}" | cut -d'@' -f2)
          echo "name=$image" >> "$GITHUB_OUTPUT"
          echo "digest=$digest" >> "$GITHUB_OUTPUT"
...
```

For the `Generate binary hashes` step, we are using the `jq` tool to parse the `artifacts` output which is one of the outputs of the [goreleaser/goreleaser-action](https://github.com/goreleaser/goreleaser-action) from the `goreleaser` job and extract the checksum file path. We are then using the `base64` tool to encode the checksum file and save it to the `hashes` output.

In essence, the artifacts output consists of the contents of the artifacts.json file that GoReleaser generates in the dist/ folder, starting from version 1.2, as explained in this v1.2 release. This file contains information regarding the artifacts produced by GoReleaser.

During the process of generating SLSA (Supply Chain Levels for Software Artifacts) provenances for both our binaries and container images, we utilize specific jq operations to extract the necessary information such as the `hashes` for the binaries and the `image` and `digest` of our container image from the artifacts.json file.

At the end of the day, you will be having a successful workflow run like this:

![image](/static/slsa-provenance-generation.png)

## Further Steps

As you can see, generating SLSA provenance for your artifacts with GoReleaser and slsa-github-generator is a straightforward process. You might be asking yourself what's next? Well, the answer is simple because we added verification steps to our workflow to show you how you can verify the SLSA provenance of your artifacts since they were signed by the slsa-github-generator and uploaded to the transparency log server (Rekor).

```yaml
...
   - name: Verify assets
        env:
          CHECKSUMS: ${{ needs.goreleaser.outputs.hashes }}
          PROVENANCE: "${{ needs.binary-provenance.outputs.provenance-name }}"
        run: |
          set -euo pipefail
          checksums=$(echo "$CHECKSUMS" | base64 -d)
          while read -r line; do
              fn=$(echo $line | cut -d ' ' -f2)
              echo "Verifying $fn"
              slsa-verifier verify-artifact --provenance-path "$PROVENANCE" \
                                            --source-uri "github.com/$GITHUB_REPOSITORY" \
                                            --source-tag "$GITHUB_REF_NAME" \
                                            "$fn"
          done <<<"$checksums"


  - name: Verify image
        env:
          IMAGE: ${{ needs.goreleaser.outputs.image }}
          DIGEST: ${{ needs.goreleaser.outputs.digest }}
        run: |
          slsa-verifier verify-image "$IMAGE@DIGEST" \
             --source-uri "github.com/$GITHUB_REPOSITORY" \
             --source-tag "$GITHUB_REF_NAME"
```

> _[slsa-verifier](https://github.com/slsa-framework/slsa-verifier) is a tool for verifying SLSA provenance that was generated by CI/CD builders. slsa-verifier verifies the provenance by verifying the cryptographic signatures on provenance to make sure it was created by the expected builder (default to GitHub CI/CD) and the source repository the artifact was built from._

> _[cosign](https://github.com/sigstore/cosign) allows developers to sign artifacts with digital signatures, ensuring the authenticity and integrity of the artifacts. It also enables users to verify signatures on artifacts to confirm that they haven't been tampered with._

Both cosign and slsa-verifier play crucial roles in enhancing the security and trustworthiness of software supply chains, particularly in containerized and cloud-native application development. To get the latest information and updates on these tools, it's recommended to refer to their respective documentation and GitHub repositories or official websites.

## Conclusion

In this blog post, we explored how GoReleaser can help you generate SLSA provenance for your artifacts and how you can leverage the slsa-github-generator to automate the process. We also discussed the importance of software supply chain security and how SLSA provenance can help you enhance the security and trustworthiness of your software supply chain. We hope that this blog post has been helpful in understanding how GoReleaser can help you generate SLSA provenance for your artifacts and how you can leverage the slsa-github-generator to automate the process. If you have any questions or feedback, please feel free to reach out to us on GoReleaser discord channel. We would love to hear from you!
