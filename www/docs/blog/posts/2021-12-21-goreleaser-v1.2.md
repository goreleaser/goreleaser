---
date: 2021-12-21
slug: goreleaser-v1.2
categories:
  - announcements
authors:
  - caarlos0
---

# Announcing GoReleaser v1.2 — Santa/5 year anniversary edition

GoReleaser v1.2 is out — likely be the last feature release of 2021. 
It also marks the first 5 years since its [first commit](https://github.com/goreleaser/goreleaser/commit/8b63e6555be45234c4c2a69576ca2ddab705302c).
It comes packed with some great features and fixes by several people!

<!-- more -->

![Christmas GoReleaser gopher!](https://carlosbecker.com/posts/goreleaser-v1.2/a49c5832-6576-42ce-989f-f717cd8096f1.png)

Here are some highlights:

1. GoReleaser now generates a `dist/artifacts.json` file — this might help integrating with other tools (e.g. you can `jq` it to find stuff). [Link](https://github.com/goreleaser/goreleaser/commit/ecb800aef7723d58f4521d4cb457a972b019ba92).
2. We now have a ["Common Errors"](https://goreleaser.com/errors/dirty/) section in our docs. This should help troubleshooting common issues and provide more context to why some errors happen. More errors should be added as we evolve. Links: [1](https://github.com/goreleaser/goreleaser/commit/62da2dbe1396aa1e423ac41feeb12f74dbe8ac29), [2](https://github.com/goreleaser/goreleaser/commit/73867736a5ddeb23ac4767cc541395e7d61d32bd), [3](https://github.com/goreleaser/goreleaser/commit/8c06005bc66ff3435bd9bee32a36ebabf685cd41).
3. Docker images are no longer listed in the changelog. It was mostly noise, making its way between the changelog and the footer, which in turn makes having custom release notes a bit harder. Users can still add them using the `footer` option. [Link](https://github.com/goreleaser/goreleaser/commit/30ff48a5a69f2441c7f4d12264c3c813e77d3467).
4. Auto-refresh checksums — now if you sign your binaries with something that actually changes the file, checksums will be regenerated and thus correct. [Link](https://github.com/goreleaser/goreleaser/commit/cbcdd41f975b29bea58b8125fee852105ff7fe88).
5. Improved universal binaries usage on Homebrew taps, Gofish rigs and Krew manifests. [Link](https://github.com/goreleaser/goreleaser/commit/e8c8a2832f42569071ff2a2d2970c1ffc7c71c96).
6. SBOM generation — using [Syft](https://github.com/anchore/syft) by default. GoReleaser v1.2 itself is now publishing its own sBOMs! [Link](https://github.com/goreleaser/goreleaser/commit/bfdec808aba208cfdedeb3bef0a16255bf1d87b3).
7. Better support for [cosign](https://github.com/sigstore/cosign)'s keyless signing. Links: [1](https://github.com/goreleaser/goreleaser/commit/7c2a93cfaa9fb5e6b0d8c1bf01a97cb5903ea7b8), [2](https://github.com/goreleaser/goreleaser/commit/994cbb47c3c6d38af15c88c712bd486a126ec4cd), [3](https://github.com/goreleaser/goreleaser/commit/505888f41be5308eb7d5c6fb25df82a1bda4cc1a).
8. Git tag annotations are now available as template variables. Links: [1](https://github.com/goreleaser/goreleaser/commit/9b9eef04a2d1e5974d6d3e2c21048b3b2c7f37f8), [2](https://github.com/goreleaser/goreleaser/commit/6ea7fb792a09525eab6089841a9fcd03e5991e35), [3](https://github.com/goreleaser/goreleaser/commit/f01c60026ce6320447736a9e562af85bbf649562).
9. Improved debug log output. [Link](https://github.com/goreleaser/goreleaser/commit/a965789203f1d64de6856a1d5b4169d32f0b06df).

You can also see the full changelog **[here](https://github.com/goreleaser/goreleaser/releases/tag/v1.2.0)**.

## **Other news**

### **Supply Chain**

A common subject in the recent releases has been supply chain security.
You can notice that by the recent improvements and collaboration with [Sigstore's Cosign](https://github.com/sigstore/cosign), and the SBOM generation feature with [Anchore's Syft](https://github.com/anchore/syft).

We intend to keep improving in this area in order to make it easier for everyone to sign their work and publish SBOMs.
Here's to safer internet in the future!

{{< img caption="Here's to safer internet in the future!" src="c3595591-87bf-4e52-baba-a6bd8089a279.png" >}}

### **Houston, we have a blog!**

We now have an official [blog](https://blog.goreleaser.com)!

Before I was [posting everything here in my personal blog](https://carlosbecker.com/tags/goreleaser/), and will likely still post some things here, but here the idea is to collaborate with all GoReleaser contributors to post news in the _official blog_.

### **Community calls and YouTube channel**

We have also scheduled our first community call! Feel free to join and suggest topics. [Link](https://github.com/goreleaser/community/pull/2).

The call should be streamed in our [YouTube channel](https://www.youtube.com/channel/UCxg5N16FKrTa4Cees434pbw). Feel free to subscribe! 😀

## **Next steps**

We'll continue to evolve GoReleaser to make it even better and easier to do the best thing possible.

If you'd like to participate, stay tuned for our first community call in January!

See y'all next year!
