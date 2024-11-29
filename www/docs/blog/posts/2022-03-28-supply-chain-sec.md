---
date: 2022-03-28
slug: supply-chain-security
categories:
  - tutorials
authors:
  - developerguy
---

# GoReleaser And Software Supply Chain Security

Before talking about the security of the software supply chains, we should mention what should come to our minds first when we are talking about software supply chains.
In most basic terms, you can think of **software supply chains are anything that's needed to deliver your product — including all the components you use**, for example, **your codebase**, **packages**, ** libs, your CI/CD pipeline**, **third-party services you use**, **anything that goes into or affects your code from development to gets deployed into production systems.**

<!-- more -->

![[https://security.googleblog.com/2021/06/introducing-slsa-end-to-end-framework.html](https://security.googleblog.com/2021/06/introducing-slsa-end-to-end-framework.html)](https://cdn-images-1.medium.com/max/2000/1*fYWJfKAY5tdAvTMJnbsQPQ.png)[https://security.googleblog.com/2021/06/introducing-slsa-end-to-end-framework.html](https://security.googleblog.com/2021/06/introducing-slsa-end-to-end-framework.html)

The picture above, taken from [SLSA (Supply Chain Levels for Software
Artifacts)](https://slsa.dev), **is a security framework**, **a check-list of
standards and controls** to **prevent tampering**, **improve the integrity**,
and secure packages and infrastructure in your projects, businesses, or enterprises, shows us the importance of protecting our workflows that delivers software we built to the customer because there are many places that attackers can attack and gain access to our system.

Unfortunately, the **new threats** in software development are not only related to the specific company itself. Thanks to [CNCF Security Technical Advisory Group](https://github.com/cncf/tag-security), they made a repository to list all the companies' compromises against supply chain attacks. Trends show that [supply chain attacks are increasing](https://www.sonatype.com/hubfs/Q3%202021-State%20of%20the%20Software%20Supply%20Chain-Report/SSSC-Report-2021_0913_PM_2.pdf?hsLang=en-us) at an **exponential rate of 4–5x per year**, with several thousand last year, the most common being related to dependency confusion or [typosquatting](https://sysdig.com/blog/malicious-python-libraries-jeilyfish-dateutil/), followed by **malicious source code injection**, so, with the rise of software supply chain attacks, it becomes more critical to secure our software supply chains.

![[https://www.memesmonkey.com/images/memesmonkey/43/43b3e5ab9f90a6266a163278c025cba5.jpeg](https://www.memesmonkey.com/images/memesmonkey/43/43b3e5ab9f90a6266a163278c025cba5.jpeg)](https://cdn-images-1.medium.com/max/2000/1*tqXzGy9XAM90BUN7cilkiw.jpeg)[https://www.memesmonkey.com/images/memesmonkey/43/43b3e5ab9f90a6266a163278c025cba5.jpeg](https://www.memesmonkey.com/images/memesmonkey/43/43b3e5ab9f90a6266a163278c025cba5.jpeg)

Securing the software supply chain is not an easy task.
So many people and organizations have already working on this specific problem, but luckily we have some tools to make that process a bit easier to deal with for us.

Today, we'll be talking about the GoReleaser project and its features that can help us mitigate the risk of compromises in software supply chains. Then, at the end of this guide, we'll be demoing everything we talked about to give more practical information about how you can make the supply chain more secure for your Go projects by using GoReleaser.

[GoReleaser](https://goreleaser.com/intro/) is a release automation tool for Go projects.
The goal is to simplify the build, release, and publish steps while providing variant customization options for all steps. **Although GoReleaser is built with CI tools in mind**, you only need to download and execute it in your build script. But, of course, you can still install it locally if you want. This is because many projects have already been using GoReleaser for a long time in the open-source world. Also, with the announcement [GitHub Actions](https://github.com/features/actions) platform, GoReleaser's popularity increased, so it takes firm steps towards becoming a defacto standard for releasing Go projects, especially in the GitHub ecosystem.

**The first step in securing your supply chain is to create an inventory of the software and libraries being used during your build and deployment cycle.**

This is where an [SBOM](https://www.linuxfoundation.org/blog/what-is-an-sbom/) comes into the picture.

**More technically**, An SBOM is a structured list of components, modules, and libraries that are included in a given piece of software. However, there are many different meanings of an SBOM in the ecosystem. In the most basic form, think of them as a list of ingredients that evolves throughout the software development lifecycle as you add new code or components. Knowing about our software's dependencies, we can first determine if we are impacted when a new security vulnerability is found. If so, we can apply security patches to mitigate the risk of that vulnerability before affecting our software.

Organizations can and do create and publish a software bill of materials in several different formats. In addition to these common formats, several methods are explicitly designed for delivering SBOMs, including [SPDX (Software Package Data Exchange)](https://spdx.dev), [Software Identification (SWID) Tags](https://csrc.nist.gov/projects/Software-Identification-SWID), and [Cyclone DX](https://cyclonedx.org), many open-source tools exist, such as [Syft](https://github.com/anchore/syft) from [Anchore](https://anchore.com), which is what we are going to talk about in this blog post, [kubernetes-sigs/bom](https://github.com/kubernetes-sigs/bom) from [Kubernetes SIG Release](https://github.com/kubernetes/sig-release), [cyclonedx-cli](https://github.com/CycloneDX/cyclonedx-cli) from [CycloneDX](https://cyclonedx.org), [spdx-sbom-generator](https://github.com/opensbom-generator/spdx-sbom-generator), [Tern](https://github.com/tern-tools/tern), and many more...

Let's look at what we have to do successfully to generate an SBOM in GoReleaser using Syft. We'll be setting up the demo on the GitHub Actions platform so that some examples might be specific to that platform.

> 🚨 **TLDR**; you can find all the source code what we are going to show you as an example on GitHub [here](https://github.com/goreleaser/supply-chain-example).

**Since GoReleaser uses Syft by calling it's binary**, we ensure that Syft binary exists before running GoReleaser. We can download the binary and move it to the executable's path while running the job, but there is a better way of doing this, [anchore/sbom-action](https://github.com/anchore/sbom-action). A sbom-action is a GitHub Action for creating a software bill of materials using Syft, but we can use its sub-action called [anchore/sbom-action/download-syft](https://github.com/anchore/sbom-action/blob/main/download-syft/action.yml) to download the executable only.

To install Syft, you need to add the following line to our GitHub Action workflow.

```yaml
- uses: anchore/sbom-action/download-syft@v0.7.0 # installs syft
```

Next, we need to add a [setting](https://goreleaser.com/customization/sbom/)
specific to configure SBOM generation to the GoReleaser configuration file
`.goreleaser.yml`.
I said configure because GoReleaser's SBOM generation support is highly configurable.
After all, it accepts commands to be run to generate an SBOM and arguments that will pass to the command, which makes GoReleaser can work with any SBOM generation tool, as we mentioned earlier, seamlessly.

```yaml
# creates SBOMs of all archives and the source tarball using syft
# https://goreleaser.com/customization/sbom
# Two different sbom configurations need two different IDs
sboms:
  - id: archive
    artifacts: archive
  - id: source
    artifacts: source
```

When you do not specify any command, it will use syftas a command by default, as you can see [here](https://github.com/goreleaser/goreleaser/blob/7671dab291483b2733e871abff379d07e74dfc6c/internal/pipe/sbom/sbom.go#L64-L73).

GoReleaser lets you cross-compile your Go binaries and package them in various formats, including container images or tarballs.
Then, using some public services such as GitHub Releases and DockerHub to distribute them to customers or production systems
With the rise of software supply chain attacks, ensuring the integrity between the artifacts (container images, blobs, etc.) we produce and consume becomes more critical.
**Integrity** means ensuring an artifact is what it says it is.
It also indicates the artifact we consume has not been tampered with since it was created and comes from a valid source, a trusted identity (a public key, a person, a company, or some other entity).
Leveraging this workflow gives your users confidence that the container images from their container registry were the trusted code you built and published.
One of the best ways of checking the integrity of an artifact and ensuring that if the artifact has been tampered with is to use a utility called [cosign](https://github.com/sigstore/cosign) from [Sigstore](https://sigstore.dev).

Sigstore is an open-source security project now sponsored by the [OpenSSF](https://openssf.org), **the Open Software Security Foundation**, allowing developers to build, distribute, and verify signed software artifacts securely.
**Sigstore** provides a cosign tool, enabling developers to build, distribute,
and verify signed software artifacts securely.

Cosign supports **several types of signing keys**, such as **text-based keys**, **cloud KMS-based keys** or **even keys generated on hardware tokens**, and **Kubernetes Secrets**, which can all be generated directly with the tool itself, and also supports adding **key-value annotations** to the signature.

Luckily, GoReleaser has built-in support for signing blobs, container images by using cosign.
The same rule for Syft applies here, too.
GoReleaser uses cosign by calling it's binary, which means that we should ensure that cosign binary exists before running GoReleaser.
Thanks to the [cosign-installer](https://github.com/sigstore/cosign-installer), a GitHub Action lets you download cosign binary.

To install cosign, you need to add the following line to our GitHub Action workflow.

```yaml
- uses: sigstore/cosign-installer@v2.1.0 # installs cosign
```

> To install cosign into your environment, please follow the installation link
> from [official website](https://docs.sigstore.dev/cosign/system_config/installation/).
> But if you are Mac user, you can start installing cosign via HomeBrew 👇
>
> ```bash
> $ brew install cosign
> ```

You can start signing your artifacts by creating public/private key pairs with the **generate-key-pair** command. Then, you need to run the **sign** command with the private key you generated. But in today's blog post, we'll be talking about a unique concept in cosign called _Keyless Signing_, which means that we no longer need to generate public/private key pairs.

> For more background on **"keyless signing"**, see blog posts on the Chainguard blog on [Fulcio](https://www.chainguard.dev/unchained/a-fulcio-deep-dive) and [keyless signing with EKS](https://www.chainguard.dev/unchained/zero-friction-keyless-signing-with-kubernetes).

It's important to note that another part of sigstore is [Fulcio](https://github.com/sigstore/fulcio),
a root CA that issues signing certificates from OIDC tokens, and [Rekor](https://github.com/sigstore/rekor),
a transparency log for certificates issued by Fulcio. In October, we announced that
[Actions runs can get OIDC tokens from GitHub for use with cloud providers](https://github.blog/changelog/2021-10-27-github-actions-secure-cloud-deployments-with-openid-connect/), including the public Fulcio and Rekor servers run by the sigstore project. You can sign your container images with the GitHub-provided OIDC token in Actions without provisioning or managing your private key.
This is critically what makes signing so easy.

> [https://github.blog/2021-12-06-safeguard-container-signing-capability-actions/](https://github.blog/2021-12-06-safeguard-container-signing-capability-actions/)

To enable signing container images in GoReleaser, all you need to do is add these lines below.

> Keyless signing is still an experimental feature in cosign, so, we should use a special environment variable to enable that support named `COSIGN_EXPERIMENTAL`.

```yaml
# signs our docker image
# https://goreleaser.com/customization/docker_sign
docker_signs:
  - cmd: cosign
    env:
      - COSIGN_EXPERIMENTAL=1
    artifacts: images
    output: true
    args:
      - "sign"
      - "${artifact}"
```

On the other hand, you must add these lines below to enable signing container images in GoReleaser.

```yaml
# signs the checksum file
# all files (including the sboms) are included in the checksum, so we don't need to sign each one if we don't want to
# https://goreleaser.com/customization/sign
signs:
  - cmd: cosign
    env:
      - COSIGN_EXPERIMENTAL=1
    certificate: "${artifact}.pem"
    args:
      - sign-blob
      - "--output-certificate=${certificate}"
      - "--output-signature=${signature}"
      - "${artifact}"
    artifacts: checksum
    output: true
```

Once you have all of these, you will end up having something like the following picture for your release:

![](https://cdn-images-1.medium.com/max/2004/1*cXC_RXowFPRJEIW41olNlg.png)

Also, a successful release pipeline:

![](https://cdn-images-1.medium.com/max/5084/1*LUmE7iOj-HLkYT-yGoJMnQ.png)
[A GitHub Actions run](https://github.com/goreleaser/goreleaser-example-supply-chain/actions/workflows/release.yml)

If you verify the container image you pushed to the ghcr.io, a **verify** command of cosign might help you verify the image's signature.

```bash
$ COSIGN_EXPERIMENTAL=1 cosign verify ghcr.io/goreleaser/supply-chain-example:v1.2.0
```

If you verify the blob, checksums.txt, in this case, you need to download the signature, the certificate, and checksums.txt itself first.

```bash
$ COSIGN_EXPERIMENTAL=1 cosign verify-blob \
  --cert checksums.txt.pem \
  --signature checksums.txt.sig
  checksums.txt \
tlog entry verified with uuid: "e42743bbbc1d06058ff7705a00bdf5046d920ede73e1fec7f313d19f5f3513b8" index: 977012
Verified OK
```

## Conclusion

GoReleaser always cares about the security of the artifacts it produces this is why it integrates with tools like cosign, Syft, etc., to mitigate the risks happening in software supply chains. As you can see from the examples we gave, it does that effortlessly by simply adding a bunch of new settings to your GoReleaser configuration file, which all happens behind the scenes without making it even more complex.

> 🍭 BONUS: Another important topic that gives you a confidence about the software’s integrity is [Reproducible Builds](http://reproducible-builds.org), are a set of software development practices that create an independently-verifiable path from source to binary code, thanks to [Carlos A. Becker](https://caarlos0.dev/), wrote a blogpost to explain it in detail, so, please do not forget to checkout it his blogpost to learn more about how GoReleaser can help you to achieve reproducibility👇
>
> [Here's the link](https://medium.com/goreleaser/reproducible-build-with-goreleaser-6de2763458a5).
