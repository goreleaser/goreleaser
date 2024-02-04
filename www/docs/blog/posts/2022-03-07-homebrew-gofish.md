---
date: 2022-03-07
slug: homebrew-gofish
categories:
  - tutorials
authors:
  - dirien
---

# GoReleaser: How To Distribute Your Binaries With Homebrew Or GoFish

This article is going to be a quick bite (or drink)! We going to discover, how
fast we can create a **Homebrew** or **GoFish** deployment of our binaries with
the help of **GoReleaser**.

<!-- more -->

But first, let us take a look into the concepts of the two package managers:

### **Homebrew **🍺

> The Missing Package Manager for macOS (or Linux)

This statement is not from me, but from the official
[Homebrew](https://brew.sh/) website. **Homebrew** is similar to other package
managers, like [apt-get](https://wiki.debian.org/apt-get),
[aptitude](https://wiki.debian.org/Aptitude), or
[dpkg](https://wiki.debian.org/dpkg). I will not go in this article into the
details about **Homebrew**, but some terms are important to understand, as we
going to use them in our `.goreleaser.yaml` file:

**Tap:** A Git repository of packages.

**Formula**: A software package. When we want to install new programs or
libraries, we install a formula.

### GoFish 🐠

> GoFish, the Package Manager 🐠

[GoFish](https://gofi.sh/) is a cross-platform systems package manager, bringing
the ease of use of Homebrew to Linux and Windows. Same as with **Homebrew**, I
am not going into detail of **GoFish** but we need also here some understanding
of the **GoFish** terminology:

**Rig:** A git repository containing fish food.

**Food:** The package definition

### The example code

For each package manager, you should create its own GitHub repository. You can
name it as you please, but i prefer to add the meaning of the repository.

- **goreleaser-rig** for GoFish
- **goreleaser-tap** for Homebrew

Add following snippet for **GoFish** support, to your existing
`.goreleaser.yaml`:

```yaml
rigs:
  - rig:
      owner: dirien
      name: goreleaser-rig
    homepage: "https://github.com/dirien/quick-bites"
    description: "Different type of projects, not big enough to warrant a separate repo."
    license: "Apache License 2.0"
```

And for **Homebrew**, add this little snippet:

```yaml
brews:
  - tap:
      owner: dirien
      name: goreleaser-tap
    folder: Formula
    homepage: "https://github.com/dirien/quick-bites"
    description: "Different type of projects, not big enough to warrant a separate repo."
    license: "Apache License 2.0"
```

That’s all for now, and as usual with GoReleaser you can head over to the great
documentation for more advanced settings:

> [https://goreleaser.com](https://goreleaser.com/intro/)

Now run the release process and you will see this in your logs:

```
  • homebrew tap formula
           • pushing formula=Formula/goreleaser-brew-fish.rb repo=dirien/goreleaser-tap
  • gofish fish food cookbook
           • pushing food=Food/goreleaser-brew-fish.lua repo=dirien/goreleaser-rig
```

Perfect! Everything works as expected.

We can check the content on the GitHub UI.

![](https://cdn-images-1.medium.com/max/5964/1*O2zfXri4yrmo_GN3clczow.png)

![](https://cdn-images-1.medium.com/max/6020/1*1TVV84tYM1staeDcifH7tw.png)

### Installation

Now we can add the tap and the rig on our clients

**Homebrew**

```bash
brew tap dirien/goreleaser-tap
brew install goreleaser-brew-fish
```

**GoFish**

```bash
gofish rig add https://github.com/dirien/goreleaser-rig
gofish install github.com/dirien/goreleaser-rig/goreleaser-brew-fish
```

### The End

Now you can distribute this tap or rig repositories and everybody can install your projects via this package manager.

![](https://cdn-images-1.medium.com/max/2560/0*prjIhehAsUYTBaLx.jpg)

### The Code

You can find the demo code in my repository, to see some more details:
[dirien/quick-bites](https://github.com/dirien/quick-bites/tree/main/goreleaser-brew-fish).
